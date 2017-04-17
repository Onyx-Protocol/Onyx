import React from 'react'
import { chainClient } from 'utility/environment'
import { actions as appActions } from 'features/app'
import { push } from 'react-router-redux'
import TokenCreateModal from './components/TokenCreateModal'

// Given a list of policies, create a grant for
// all policies that are truthy, and delete any
// outstanding grants for policies that are not.
const setPolicies = (body, policies) => {
  const promises = []

  for (let key in policies) {
    const grant = {
      ...body,
      policy: key
    }

    promises.push(policies[key] ?
      chainClient().accessControl.create(grant) :
      chainClient().accessControl.delete(grant)
    )
  }

  return Promise.all(promises)
}

export default {
  fetchItems: () => {
    return (dispatch) => {
      const tokens = []

      return Promise.all([
        chainClient().accessControl.list(),
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
      if (!Object.values(data.policies).some(policy => policy == true)) {
        return Promise.reject({_error: 'You must specify one or more policies'})
      }

      return chainClient().accessTokens.create({
        id: body.guardData.id,
        type: 'client', // TODO: remove me when deprecated!
      }).then(tokenResp =>
        setPolicies(body, data.policies).then(grantResp => {
          dispatch(appActions.showModal(
            <TokenCreateModal token={tokenResp.token}/>,
            appActions.hideModal
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
    const body = {
      guardType: 'x509',
      guardData: {subject: {}},
      policy: 'client-readwrite'
    }

    for (let index in data.subject) {
      const field = data.subject[index]
      body.guardData.subject[field.key] = field.value
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
