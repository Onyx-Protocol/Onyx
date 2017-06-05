import { client } from '../core'
import { FETCH } from './constants'
import { Item } from './types'

export const fetch = () => {
  let items: Item[] = []
  return (dispatch, getState) => {
    return client.assets.queryAll({
      filter: "is_local='yes'",
      pageSize: 100
    }, function(item, next) {
      items.push(item)
      next();
    }).then(() => {
      dispatch({
        type: FETCH,
        items: items
      })
    }).catch(err => {
      console.log(err)
      throw err
    })
  }
}
