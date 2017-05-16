// external imports
const chain = require('chain-sdk')

// internal imports
import * as types from './types'

let url: string
const isProd: boolean = process.env.NODE_ENV === 'production'
if (isProd) {
  url = window.location.origin
} else {
  // Used to proxy requests from the client to core.
  url = process.env.API_URL || 'http://localhost:8081/api'
}

const dashboardState = importState()
const accessToken = dashboardState.core && dashboardState.core.clientToken
export const client = new chain.Client({
  url,
  accessToken
})
export const signer = new chain.HsmSigner()

// Prefixes the redux router route during production builds.
export const prefixRoute = (route: string): string => {
  if (isProd) {
    return "/ivy" + route
  }
  return route
}

// Parses an error from Chain Core
export const parseError = (err) => {
  if (err === undefined) {
    return ''
  }

  const body = err.body
  if (err.code === 'CH706' && body && body.data) {
    return body.data.actions.reduce((msg, action, i, arr) => {
      if (i < arr.length - 1) {
        msg += "\n"
      }
      return msg + action.code + ": " + action.message
    }, "")
  }
  return err.message
}

// Imports the dashboard's redux state from localStorage.
// This is used to retrieve the client access token, if it exists.
// Taken directly from dashboard code.
function importState() {
  let state
  try {
    state = localStorage.getItem('reduxState')
  } catch (err) { /* localstorage not available */ }

  if (!state) return {}

  try {
    return JSON.parse(state)
  } catch (_) {
    return {}
  }
}

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
