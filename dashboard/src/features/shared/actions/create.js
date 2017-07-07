import { chainClient } from 'utility/environment'
import { parseNonblankJSON } from 'utility/string'
import { push } from 'react-router-redux'
import uuid from 'uuid'

export default function(type, options = {}) {
  const listPath = options.listPath || `/${type}s`
  const createPath = options.createPath || `${listPath}/create`
  const created = (param) => ({ type: `CREATED_${type.toUpperCase()}`, param })

  return {
    showCreate: push(createPath),
    created,
    submitForm: (data) => {
      const clientApi = options.clientApi ? options.clientApi() : chainClient()[`${type}s`]
      let promise = Promise.resolve()

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

      if (data.xpubs) {
        data.rootXpubs = []
        data.xpubs.forEach(key => {
          if (key.type == 'generate') {
            promise = promise
              .then(() => {
                const alias = (key.value || '').trim()
                  ? key.value.trim()
                  : (data.alias || 'generated') + '-' + uuid.v4()

                return chainClient().mockHsm.keys.create({alias})
              }).then(newKey => {
                data.rootXpubs.push(newKey.xpub)
              })
          } else if (key.value) {
            data.rootXpubs.push(key.value)
          }
        })
        delete data.xpubs
      }

      return function(dispatch) {
        return promise.then(() => clientApi.create(data)
          .then((resp) => {
            dispatch(created(resp))

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
