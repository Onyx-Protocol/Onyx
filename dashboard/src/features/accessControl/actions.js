import React from 'react'
import { chainClient } from 'utility/environment'
import { actions as appActions } from 'features/app'
import { push } from 'react-router-redux'
import { policyOptions, subjectFieldOptions } from './constants'
import { hasProtectedGrant } from './selectors'
import TokenCreateModal from './components/TokenCreateModal'

// Given a list of policies, create a grant for
// all policies that are truthy, and delete any
// outstanding grants for policies that are not.
const setPolicies = (body, policies) => {
  const promises = []

  for (let policy in policies) {
    if (policyOptions.find(opt => opt.value == policy).hidden) continue
    if (hasProtectedGrant(body.grants || [], policy)) continue

    const grant = {
      guardData: body.guardData,
      guardType: body.guardType,
      policy
    }

    try {
      promises.push(policies[policy] ?
        chainClient().authorizationGrants.create(grant) :
        chainClient().authorizationGrants.delete(grant)
      )
    } catch (err) {
      promises.push(Promise.reject(err))
    }
  }

  return Promise.all(promises)
}

export default {
  fetchItems: () => {
    return (dispatch) => {
      const tokens = []

      return Promise.all([
        chainClient().authorizationGrants.list(),
        chainClient().accessTokens.queryAll({}, (token, next) => {
          tokens.push(token)
          next()
        })
      ]).then(result => {
        const grants = result[0].items
        return dispatch({ type: 'RECEIVED_ACCESS_GRANTS', grants, tokens })
      })
    }
  },

  submitTokenForm: data => {
    const body = {
      guardType: 'access_token',
      guardData: data.guardData
    }

    return dispatch => {
      return chainClient().accessTokens.create({
        id: body.guardData.id,
        type: 'client', // TODO: remove me when deprecated!
      }).then(tokenResp =>
        setPolicies(body, data.policies).then(grantResp => {
          dispatch(appActions.showModal(
            'New access token created',
            <TokenCreateModal token={tokenResp.token}/>
          ))

          dispatch({ type: 'CREATED_TOKEN_WITH_GRANT', grantResp })

          dispatch(push({
            pathname: '/access-control',
            search: '?type=token',
            state: {preserveFlash: true},
          }))
        })
      ).catch(err => { throw {_error: err} })
    }
  },

  submitCertificateForm: data => {
    const fieldInfo = {}
    for (let index in subjectFieldOptions) {
      const option = subjectFieldOptions[index]
      fieldInfo[option.value] = option
    }

    const body = {
      guardType: 'x509',
      guardData: {subject: {}},
    }

    for (let index in data.subject) {
      const field = data.subject[index]

      if (field.key && fieldInfo[field.key].array) {
        const values = body.guardData[field.key] || []
        values.push(field.value)
        body.guardData.subject[field.key] = values
      } else {
        body.guardData.subject[field.key] = field.value
      }
    }

    return dispatch => {
      if (!Object.values(data.policies).some(policy => policy == true)) {
        return Promise.reject({_error: 'You must specify one or more policies'})
      }

      return setPolicies(body, data.policies).then(resp => {
        dispatch({ type: 'CREATED_X509_GRANT', resp })
        dispatch(push({
          pathname: '/access-control',
          search: '?type=certificate',
          state: {preserveFlash: true},
        }))
      }, err => { throw {_error: err} })
    }
  },

  beginEditing: id => ({
    type: 'BEGIN_POLICY_EDITING',
    id: id
  }),

  editPolicies: data => {
    const body = data.grant
    const policies = data.policies

    return dispatch =>
      setPolicies(body, policies).then(() => {
        dispatch({ type: 'END_POLICY_EDITING', id: data.grant.id, policies })
      }, err => { throw {_error: err} })
  },

  deleteToken: grant => {
    const id = grant.guardData.id
    if (!window.confirm(`Really delete access token "${id}"?`)) {
      return
    }

    return dispatch => chainClient().accessTokens.delete(id)
      .then(() => {
        dispatch({
          type: 'DELETE_ACCESS_TOKEN',
          id: grant.id,
          message: 'Token deleted.'
        })
      }).catch(err => dispatch({
        type: 'ERROR', payload: err
      }))
  }
}
