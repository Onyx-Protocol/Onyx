import chain from '../chain'
import { context } from '../utility/environment'
import { push } from 'react-router-redux'

export default function(type, options = {}) {
  const listPath   = options.listPath || `/${type}s`
  const createPath = options.createPath || `/${type}s/create`

  return {
    showCreate: () => function(dispatch) {
      dispatch(push(createPath))
    },

    submitForm: (data) => {
      const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)
      return function(dispatch) {
        let object = new chain[className](data)

        object.create(context)
          .then(() => {
            options.resetAction(dispatch)
            dispatch(push(listPath))
          }).catch((err) => {
            console.log(err)
          })
      }
    }
  }
}
