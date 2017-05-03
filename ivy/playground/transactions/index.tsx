import {
  client,
  signer
} from '../util'

import {
  Action
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

export const createSpendingTx = (actions: Action[]): Promise<string> => {
  console.log("actions", actions)
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
          builder.spendUnspentOutput(action)
          break
        default:
          break
      }
    })
  }).catch((e) => {
    console.log(e.body)
  })
}