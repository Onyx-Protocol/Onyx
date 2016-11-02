import chain from 'chain'
import { context } from 'utility/environment'
import { parseNonblankJSON } from 'utility/string'
import actionCreator from './actionCreator'
import { push } from 'react-router-redux'
import actions from 'actions'
import uuid from 'uuid'

export default function(type, options = {}) {
  const listPath = options.listPath || `/${type}s`
  const createPath = options.createPath || `${listPath}/create`
  const created = actionCreator(`CREATED_${type.toUpperCase()}`, param => ({ param }) )

  return {
    showCreate: push(createPath),
    created,
    submitForm: (data) => {
      const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)
      let promise = new Promise((resolve) => resolve())

      if (typeof data.id == 'string')     data.id = data.id.trim()
      if (typeof data.alias == 'string')  data.alias = data.alias.trim()

      const jsonFields = options.jsonFields || []
      jsonFields.map(fieldName => {
        data[fieldName] = parseNonblankJSON(data[fieldName])
      })

      const intFields = options.intFields || []
      intFields.map(fieldName => {
        data[fieldName] = parseInt(data[fieldName])
      })

      const xpubs = data.xpubs || []
      xpubs.map(key => {
        if (key.type == 'generate') {
          promise = promise
            .then(() => {
              const alias = (key.value || '').trim()
                ? key.value.trim()
                : (data.alias || 'generated') + '-' + uuid.v4()

              return new chain.MockHsm({alias}).create(context())
            }).then(newKey => {
              data.root_xpubs = [...(data.root_xpubs || []), newKey.xpub]
            })
        } else {
          data.root_xpubs = [...(data.root_xpubs || []), key.value]
        }
      })
      delete data.xpubs

      return function(dispatch) {
        return promise.then(() => new chain[className](data).create(context())
          .then((resp) => {
            dispatch(created(resp))

            if (options.createModal) {
              dispatch(actions.app.showModal(
                options.createModal(resp),
                actions.app.hideModal()
              ))
            }

            let postCreatePath = listPath
            if (options.redirectToShow) {
              postCreatePath = `${postCreatePath}/${resp.id}`
            }

            dispatch(push({
              pathname: postCreatePath,
              state: {
                preserveFlash: true
              }
            }))
          }))
      }
    }
  }
}
