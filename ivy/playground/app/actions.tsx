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
    let selected = templates.selectors.getSelected(getState())
    if ( selected === "") {
      selected = "LockWithPublicKey"
    }
    dispatch({ type: RESET })
    dispatch(templates.actions.load(selected))
    dispatch(accounts.actions.fetch())
    dispatch(assets.actions.fetch())
  }
}

export const SEED: string = "app/SEED"

export const seed = () => {
  return (dispatch, getState) => {
    Promise.resolve().then(() => {
      const createKeys = [
        client.mockHsm.keys.create(),
        client.mockHsm.keys.create(),
        client.mockHsm.keys.create(),
        client.mockHsm.keys.create(),
        client.mockHsm.keys.create()
      ]
      return Promise.all(createKeys)
    }).then(keys => {
      keys.forEach(key => {
        signer.addKey(key.xpub, client.mockHsm.signerConnection)
      })

      const createEntities = [
        client.assets.create({
          alias: 'USD',
          rootXpubs: [keys[0].xpub],
          quorum: 1,
        }),

        client.assets.create({
          alias: 'Gold',
          rootXpubs: [keys[1].xpub],
          quorum: 1,
        }),

        client.accounts.create({
          alias: 'Escrow Agent',
          rootXpubs: [keys[2].xpub],
          quorum: 1
        }),

        client.accounts.create({
          alias: 'Bob',
          rootXpubs: [keys[3].xpub],
          quorum: 1
        }),

        client.accounts.create({
          alias: 'Alice',
          rootXpubs: [keys[4].xpub],
          quorum: 1
        })
      ]
      return Promise.all(createEntities)
    }).then(entities => {
      return client.transactions.build(builder => {
        builder.issue({
          assetAlias: 'Gold',
          amount: 10000
        })

        builder.controlWithAccount({
          accountAlias: 'Alice',
          assetAlias: 'Gold',
          amount: 10000
        })

        builder.issue({
          assetAlias: 'USD',
          amount: 10000
        })

        builder.controlWithAccount({
          accountAlias: 'Bob',
          assetAlias: 'USD',
          amount: 10000
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
