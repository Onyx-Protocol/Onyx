import { client } from '../util'

import { FETCH } from './constants'

export const fetch = () => {
  return (dispatch, getState) => {
    client.accounts.query().then(data => {
      dispatch({
        type: FETCH,
        items: data.items
      })
    }).catch(err => {
      throw err
    })
  }
}
