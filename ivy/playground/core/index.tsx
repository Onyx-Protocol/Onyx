// external imports
const chain = require('chain-sdk')

// internal imports
import * as types from './types'

let apiHost: string
const isProd: boolean = process.env.NODE_ENV === 'production'
if (isProd) {
  apiHost = window.location.origin
} else {
  apiHost = process.env.API_URL || 'http://localhost:8080/api'
}

// Prefixes the redux router route during production builds.
export const prefixRoute = (route: string): string => {
  if (isProd) {
    return "/ivy" + route
  }
  return route
}

export const client = new chain.Client({
  url: apiHost
})

export const signer = new chain.HsmSigner()

// Uses the ivy contract to lock value.
export const createLockingTx = (actions: types.Action[]): Promise<Object> => {
  return client.transactions.build(builder => {
    actions.forEach(action => {
      switch (action.type) {
        case "spendFromAccount":
          builder.spendFromAccount(action)
          break
        case "controlWithReceiver":
          builder.controlWithReceiver(action)
          break
        default:
          break
      }
    })
  }).then((tpl) => {
    tpl.signingInstructions.forEach((instruction) => {
      instruction.witnessComponents.forEach((component) => {
        component.keys.forEach((key) => {
          signer.addKey(key.xpub, client.mockHsm.signerConnection)
        })
      })
    })
    return signer.sign(tpl)
  }).then((tpl) => {
    return client.transactions.submit(tpl)
  }).then((tx) => {
    return client.unspentOutputs.query({"filter": "transaction_id=$1", "filterParams": [tx.id]})
  }).then((utxos) => {
    return utxos.items.find(utxo => utxo.purpose !== 'change')
  })
}

// Satisfies created contract and transfers value.
export const createUnlockingTx = (actions: types.Action[],
                               witness: types.WitnessComponent[],
                               mintimes,
                               maxtimes): Promise<{id: string}> => {
  return client.transactions.build(builder => {
    actions.forEach(action => {
      switch (action.type) {
        case "spendFromAccount":
          builder.spendFromAccount(action)
          break
        case "controlWithReceiver":
          builder.controlWithReceiver(action)
          break
        case "controlWithAccount":
          builder.controlWithAccount(action)
          break
        case "spendUnspentOutput":
          builder.spendAnyUnspentOutput(action)
          break
        default:
          break
      }
    })

    if (mintimes.length > 0) {
      const findMax = (currMax, currVal) => {
        if (currVal.getTime() > currMax.getTime()) {
          return currVal
        }
        return currMax
      }
      const mintime = new Date(mintimes.reduce(findMax, mintimes[0]))
      builder.minTime = new Date(mintime.setSeconds(mintime.getSeconds() + 1))
    }

    if (maxtimes.length > 0) {
      const findMin = (currMin, currVal) => {
        if (currVal.getTime() < currMin.getTime()) {
          return currVal
        }
        return currMin
      }
      const maxtime = maxtimes.reduce(findMin, maxtimes[0])
      builder.maxTime = new Date(maxtime.setSeconds(maxtime.getSeconds() - 1))
    }
  }).then((tpl) => {
    tpl.includesContract = true
    // TODO(boymanjor): Can we depend on contract being on first utxo?
    tpl.signingInstructions[0].witnessComponents = witness
    tpl.signingInstructions.forEach((instruction, idx) => {
      instruction.witnessComponents.forEach((component) => {
        if (component.keys === undefined) {
          return
        }
        component.keys.forEach((key) => {
          signer.addKey(key.xpub, client.mockHsm.signerConnection)
        })
      })
    })
    return signer.sign(tpl)
  }).then((tpl) => {
    witness = tpl.signingInstructions[0].witnessComponents
    if (witness !== undefined) {
      tpl.signingInstructions[0].witnessComponents = witness.map(component => {
        switch(component.type) {
          case "raw_tx_signature":
            return {
              type: "data",
              value: component.signatures[0]
            } as types.DataWitness
          default:
            return component
        }
      })
    }
    return client.transactions.submit(tpl)
  })
}
