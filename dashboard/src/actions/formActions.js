import chain from '../chain'
import { context } from '../utility/environment'
import { parseNonblankJSON } from '../utility/string'
import actionCreator from './actionCreator'
import { push } from 'react-router-redux'

export default function(type, options = {}) {
  const listPath   = options.listPath || `/${type}s`
  const createPath = options.createPath || `/${type}s/create`
  const created = actionCreator(`CREATED_${type.toUpperCase()}`, param => ({ param }) )

  return {
    showCreate: push(createPath),
    created,
    submitForm: (data) => {
      const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)

      const jsonFields = options.jsonFields || []
      jsonFields.map(fieldName => {
        data[fieldName] = parseNonblankJSON(data[fieldName])
      })

      return function(dispatch) {
        let object = new chain[className](data)

        return object.create(context)
          .then(() => {
            dispatch(push(listPath))
            dispatch(created())
          })
      }
    }
  }
}
