import { chainClient } from 'utility/environment'
import { push } from 'react-router-redux'

export default function(type, options = {}) {
  const updated = (param) => ({ type: `UPDATED_${type.toUpperCase()}`, param })

  return {
    updated,
    submitUpdateForm: (data, id) => {
      const clientApi = options.clientApi ? options.clientApi() : chainClient()[`${type}s`]
      let promise = Promise.resolve()

      return function(dispatch) {
        return promise.then(() => clientApi.updateTags({
          id: id,
          tags: JSON.parse(data.tags),
        }).then((resp) => {
          dispatch(updated(resp))

          dispatch(push({
            pathname: `/${type}s/${id}`,
            state: {
              preserveFlash: true
            }
          }))
        }))
      }
    }
  }
}
