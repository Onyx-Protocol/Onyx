import React from 'react'
import { chainClient } from 'utility/environment'
import { baseListActions } from 'features/shared/actions'
import { actions as appActions } from 'features/app'
import { push } from 'react-router-redux'
import TokenCreateModal from './components/TokenCreateModal'

export default {
  ...baseListActions('accessControl', {
    clientApi: () => chainClient().accessControl
  }),

  submitTokenForm: data => {
    const body = {...data}

    body.guardType = 'access_token'

    return function(dispatch) {
      return chainClient().accessTokens.create({
        id: body.guardData.id,
        type: 'client', // TODO: remove me when deprecated!
      }).then(tokenResp => {
        chainClient().accessControl.create(body).then(grantResp => {
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
      }).catch(err => Promise.reject({_error: err}))
    }
  },

  submitCertificateForm: data => {
    const subject = data.subject
    delete data.subject

    const body = {...data}
    body.guardType = 'x509'
    body.guardData = {subject: {}}
    for (var index in subject) {
      const field = subject[index]
      body.guardData.subject[field.key] = field.value
    }

    return function(dispatch) {
      return chainClient().accessControl.create(body).then(resp => {
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
