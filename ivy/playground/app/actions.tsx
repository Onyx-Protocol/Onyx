// external imports
import * as React from 'react'

// ivy imports
import accounts from '../accounts'
import assets from '../assets'
import templates from '../templates'
import contracts from '../contracts'
import { client, signer } from '../core'

export const RESET: string = "app/RESET"

export const reset = () => {
  return (dispatch, getState) => {
    let selected = templates.selectors.getSelectedTemplate(getState())
    if (selected === "" || templates.constants.INITIAL_SOURCE_MAP[selected] === undefined) {
      selected = templates.constants.INITIAL_ID_LIST[1]
    }
    dispatch({ type: RESET })
    dispatch(templates.actions.loadTemplate(selected))
    dispatch(accounts.actions.fetch())
    dispatch(assets.actions.fetch())
  }
}

export const SEED: string = "app/SEED"

export const seed = () => {
  return (dispatch, getState) => {
    return client.mockHsm.keys.create().then(key => {
      signer.addKey(key.xpub, client.mockHsm.signerConnection)
      const createEntities = [

        client.assets.create({
          alias: 'USD',
          rootXpubs: [key.xpub],
          quorum: 1,
        }),

        client.assets.create({
          alias: 'Gold',
          rootXpubs: [key.xpub],
          quorum: 1,
        }),

        client.assets.create({
          alias: 'EUR',
          rootXpubs: [key.xpub],
          quorum: 1,
        }),

        client.assets.create({
          alias: 'Acme Stock',
          rootXpubs: [key.xpub],
          quorum: 1,
        }),

        client.accounts.create({
          alias: 'FX Dealer',
          rootXpubs: [key.xpub],
          quorum: 1
        })

        client.accounts.create({
          alias: 'Escrow Agent',
          rootXpubs: [key.xpub],
          quorum: 1
        }),

        client.accounts.create({
          alias: 'Bob',
          rootXpubs: [key.xpub],
          quorum: 1
        }),

        client.accounts.create({
          alias: 'Alice',
          rootXpubs: [key.xpub],
          quorum: 1
        })
      ]
      return Promise.all(createEntities)
    }).then(entities => {
      return client.transactions.build(builder => {
        builder.issue({
          assetAlias: 'Gold',
          amount: 1000
        })

        builder.controlWithAccount({
          accountAlias: 'Alice',
          assetAlias: 'Gold',
          amount: 1000
        })

        builder.issue({
          assetAlias: 'Acme Stock',
          amount: 1000
        })

        builder.controlWithAccount({
          accountAlias: 'Bob',
          assetAlias: 'Acme Stock',
          amount: 1000
        })

        builder.issue({
          assetAlias: 'USD',
          amount: 30000
        })

        builder.controlWithAccount({
          accountAlias: 'Bob',
          assetAlias: 'USD',
          amount: 10000
        })

        builder.controlWithAccount({
          accountAlias: 'FX Dealer',
          assetAlias: 'USD',
          amount: 20000
        })

        builder.issue({
          assetAlias: 'EUR',
          amount: 30000
        })

        builder.controlWithAccount({
          accountAlias: 'Alice',
          assetAlias: 'EUR',
          amount: 10000
        })

        builder.controlWithAccount({
          accountAlias: 'FX Dealer',
          assetAlias: 'EUR',
          amount: 20000
        })
      })
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    }).then(res => {
      dispatch(accounts.actions.fetch())
      dispatch(assets.actions.fetch())
    }).catch(err =>
      process.nextTick(() => { throw err })
    )
  }
}
