import uuid from 'uuid'
import { chainClient } from 'utility/environment'
import { parseNonblankJSON } from 'utility/string'
import { push } from 'react-router-redux'
import { baseCreateActions, baseListActions } from 'features/shared/actions'

const type = 'transaction'

const list = baseListActions(type, {
  defaultKey: 'id'
})
const form = baseCreateActions(type)

function preprocessTransaction(formParams) {
  const copy = JSON.parse(JSON.stringify(formParams))
  const builder = {
    baseTransaction: copy.baseTransaction,
    actions: copy.actions,
  }

  if (builder.baseTransaction == '') {
    delete builder.baseTransaction
  }

  if (formParams.submitAction == 'generate') {
    builder.ttl = '1h' // 1 hour
  }

  for (let i in builder.actions) {
    let a = builder.actions[i]

    const intFields = ['amount', 'position']
    intFields.forEach(key => {
      const value = a[key]
      if (value) {
        if ((parseInt(value)+'') == value) {
          a[key] = parseInt(value)
        } else {
          throw new Error(`Action ${parseInt(i)+1} ${key} must be an integer.`)
        }
      }
    })

    try {
      a.referenceData = parseNonblankJSON(a.referenceData)
    } catch (err) {
      throw new Error(`Action ${parseInt(i)+1} reference data should be valid JSON, or blank.`)
    }

    try {
      a.receiver = parseNonblankJSON(a.receiver)
    } catch (err) {
      throw new Error(`Action ${parseInt(i)+1} receiver should be valid JSON.`)
    }
  }

  return builder
}

function getTemplateXpubs(tpl) {
  const xpubs = []
  tpl.signingInstructions.forEach((instruction) => {
    instruction.witnessComponents.forEach((component) => {
      component.keys.forEach((key) => {
        xpubs.push(key.xpub)
      })
    })
  })
  return xpubs
}

form.submitForm = (formParams) => function(dispatch) {
  const client = chainClient()

  const buildPromise = client.transactions.build(builder => {
    const processed = preprocessTransaction(formParams)

    builder.actions = processed.actions
    if (processed.baseTransaction) {
      builder.baseTransaction = processed.baseTransaction
    }
  })

  if (formParams.submitAction == 'submit') {
    return buildPromise
      .then(tpl => {
        getTemplateXpubs(tpl).forEach(key => {
          client.signer.addKey(key, client.mockHsm.signerConnection)
        })

        return client.transactions.sign(tpl)
      }).then(signed => client.transactions.submit(signed))
      .then(resp => {
        dispatch(form.created())
        dispatch(push({
          pathname: `/transactions/${resp.id}`,
          state: {
            preserveFlash: true
          }
        }))
      })
  }

  // submitAction == 'generate'
  return buildPromise
    .then(tpl => {
      getTemplateXpubs(tpl).forEach(key => {
        client.signer.addKey(key, client.mockHsm.signerConnection)
      })

      return client.transactions.sign({...tpl, allowAdditionalActions: true})
    })
    .then(signed => {
      const id = uuid.v4()
      dispatch({
        type: 'GENERATED_TX_HEX',
        generated: {
          id: id,
          hex: signed.rawTransaction,
        },
      })
      dispatch(push(`/transactions/generated/${id}`))
    })
}

export default {
  ...list,
  ...form,
}
