import React from 'react'
import { chainClient } from 'utility/environment'
import { baseListActions } from 'features/shared/actions'
import { actions as appActions } from 'features/app'
import { push } from 'react-router-redux'
import TokenCreateModal from './components/TokenCreateModal'

const baseActions = baseListActions('accessControl', {
  clientApi: () => chainClient().accessControl
})

// Given a list of policies, create a grant for
// all policies that are truthy, and revoke any
// outstanding grants for policies that are not.
const setPolicies = (body, policies) => {
  const promises = []

  for (var key in policies) {
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
      const promise = chainClient().accessControl.list()

      promise.then(
        (param) => dispatch({
          type: 'RECEIVED_ACCESSCONTROL_ITEMS',
          param,
        })
      )

      return promise
    }
  },

  deleteItem: baseActions.deleteItem,

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
            <TokenCreateModal token={tokenResp.token}/>,
            appActions.hideModal
          ))

          dispatch({ type: 'CREATED_ACCESSTOKEN', grantResp })

          dispatch(push({
            pathname: '/access-control',
            search: '?type=token',
            state: {preserveFlash: true},
          }))
        })
      ).catch(err => Promise.reject({_error: err}))
    }
  },

  submitCertificateForm: data => {
    const body = {
      guardType: 'x509',
      guardData: {subject: {}},
      policy: 'client-readwrite'
    }

    for (var index in data.subject) {
      const field = data.subject[index]
      body.guardData.subject[field.key] = field.value
    }

    return dispatch => {
      return setPolicies(body, data.policies).then(resp => {
        dispatch({ type: 'CREATED_ACCESSX509', resp })
        dispatch(push({
          pathname: '/access-control',
          search: '?type=certificate',
          state: {preserveFlash: true},
        }))
      }, err => Promise.reject({_error: err}))
    }
  },

  revokeGrant: grant => {
    if (!window.confirm('Really delete access grant?')) {
      return
    }

    return dispatch => chainClient().accessControl.delete(grant)
      .then(() => {
        dispatch({
          type: 'DELETE_ACCESSCONTROL',
          id: grant.id,
          message: 'Grant revoked.'
        })
      }).catch(err => dispatch({
        type: 'ERROR', payload: err
      }))
  }
}
