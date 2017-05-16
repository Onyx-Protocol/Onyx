import { client } from '../core'
import { FETCH } from './constants'

export const fetch = () => {
  return (dispatch, getState) => {
    return client.assets.query({
      filter: "is_local='yes'"
    }).then(data => {
      dispatch({
        type: FETCH,
        items: data.items
      })
    }).catch(err => {
      throw err
    })
  }
}
