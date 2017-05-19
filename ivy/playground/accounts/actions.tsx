// external imports
import * as React from 'react'

// ivy imports
import { client } from '../core'

// internal imports
import { FETCH } from './constants'
import { Item } from './types'
import { seed } from '../app/actions'

export const fetch = () => {
  return (dispatch, getState) => {
    let items: Item[] = []
    const type = FETCH
    client.accounts.queryAll({
      pageSize: 100
    }, function(item, next) {
      items.push(item)
      next();
    }).then(() => {
      if (items.length == 0) {
        dispatch(seed())
      }
      const getBalances = items.map(item => {
        return client.balances.query({
          filter: 'account_alias=$1',
          filterParams: [item.alias]
        })
      })
      return Promise.all(getBalances)
    }).then(balances => {
      return dispatch({
        type,
        items,
        balances
      })
    }).catch(err => {
      throw err
    })
  }
}
