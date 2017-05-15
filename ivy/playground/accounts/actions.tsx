// external imports
import * as React from 'react'

// ivy imports
import { client } from '../core'

// internal imports
import { FETCH } from './constants'

export const fetch = () => {
  return (dispatch, getState) => {
    let items
    const type = FETCH
    client.accounts.query().then(data => {
      items = data.items
      const getBalances = data.items.map(item => {
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
