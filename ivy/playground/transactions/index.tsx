import {
  client,
  signer
} from '../util'

import {
  Action,
  DataWitness,
  WitnessComponent
} from './types'

export function createFundingTx(actions: Action[]): Promise<Object> {
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

export const createSpendingTx = (actions: Action[], witness: WitnessComponent[]): Promise<Object> => {
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
  }).then((tpl) => {
    // there should only be one
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
            } as DataWitness
          default:
            return component
        }
      })
    }
    return client.transactions.submit(tpl)
  })
}
