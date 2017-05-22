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
    const accountsList: { alias: string }[] = []
    const accountsPromise = client.accounts.queryAll({
      pageSize: 100
    }, function(item, next) {
      accountsList.push(item)
      next();
    })

    const assetsList: { alias: string }[] = []
    const assetsPromise = client.assets.queryAll({
      filter: "is_local='yes'",
      pageSize: 100
    }, function(item, next) {
      assetsList.push(item)
      next();
    })
    return Promise.all([accountsPromise, assetsPromise]).then(() => {
      const accountsMap: { [s: string]: { alias: string } } = accountsList.reduce((map, account) => {
        if (map[account.alias]) {
          return map
        }
        return {
          ...map,
          [account.alias]: true
        }
      }, {})

      const assetsMap: { [s: string]: { alias: string } } = assetsList.reduce((map, asset) => {
        if (map[asset.alias]) {
          return map
        }
        return {
          ...map,
          [asset.alias]: true
        }
      }, {})
      return Promise.resolve([accountsMap, assetsMap])
    }).then(maps => {
      const accountsMap = maps[0]
      const assetsMap = maps[1]
      return client.mockHsm.keys.create().then(key => {
        signer.addKey(key.xpub, client.mockHsm.signerConnection)
        const createEntities: Promise<Object>[] = []

        if (!assetsMap['USD']) {
          createEntities.push(
            client.assets.create({
              alias: 'USD',
              rootXpubs: [key.xpub],
              quorum: 1,
            })
          )
        }

        if (!assetsMap['Gold']) {
          createEntities.push(
            client.assets.create({
              alias: 'Gold',
              rootXpubs: [key.xpub],
              quorum: 1,
            })
          )
        }

        if (!assetsMap['EUR']) {
          createEntities.push(
            client.assets.create({
              alias: 'EUR',
              rootXpubs: [key.xpub],
              quorum: 1,
            })
          )
        }

        if (!assetsMap['Acme Stock']) {
          createEntities.push(
            client.assets.create({
              alias: 'Acme Stock',
              rootXpubs: [key.xpub],
              quorum: 1,
            })
          )
        }

        if (!accountsMap['FX Dealer']) {
          createEntities.push(
            client.accounts.create({
              alias: 'FX Dealer',
              rootXpubs: [key.xpub],
              quorum: 1,
            })
          )
        }

        if (!accountsMap['Escrow Agent']) {
          createEntities.push(
            client.accounts.create({
              alias: 'Escrow Agent',
              rootXpubs: [key.xpub],
              quorum: 1
            })
          )
        }

        if (!accountsMap['Bob']) {
          createEntities.push(
            client.accounts.create({
              alias: 'Bob',
              rootXpubs: [key.xpub],
              quorum: 1
            })
          )
        }

        if (!accountsMap['Alice']) {
          createEntities.push(
            client.accounts.create({
              alias: 'Alice',
              rootXpubs: [key.xpub],
              quorum: 1
            })
          )
        }
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
        issuance.signingInstructions.forEach((instruction) => {
          instruction.witnessComponents.forEach((component) => {
            component.keys.forEach((key) => {
              signer.addKey(key.xpub, client.mockHsm.signerConnection)
            })
          })
        })
        return signer.sign(issuance)
      }).then(signed => {
        return client.transactions.submit(signed)
      }).then(res => {
        const type = SEED
        dispatch({ type })
        dispatch(accounts.actions.fetch())
        dispatch(assets.actions.fetch())
      }).catch(err =>
        process.nextTick(() => { throw err })
      )
    })
  }
}
