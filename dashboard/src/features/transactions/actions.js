import uuid from 'uuid'
import { chainClient, chainSigner } from 'utility/environment'
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

    // HACK: Check for retire actions and replace with OP_FAIL control programs.
    // TODO: update JS SDK to support Java SDK builder style.
    if (a.type == 'retire_asset') {
      a.type = 'control_program'
      a.controlProgram = '6a' // OP_FAIL hex byte
    }

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
  const buildPromise = chainClient().transactions.build(builder => {
    const processed = preprocessTransaction(formParams)

    builder.actions = processed.actions
    if (processed.baseTransaction) {
      builder.baseTransaction = processed.baseTransaction
    }
  })

  if (formParams.submitAction == 'submit') {
    return buildPromise
      .then(tpl => {
        const signer = chainSigner()

        getTemplateXpubs(tpl).forEach(key => {
          signer.addKey(key, chainClient().mockHsm.signerConnection)
        })

        return signer.sign(tpl)
      }).then(signed => chainClient().transactions.submit(signed))
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
      const signer = chainSigner()

      getTemplateXpubs(tpl).forEach(key => {
        signer.addKey(key, chainClient().mockHsm.signerConnection)
      })

      return signer.sign({...tpl, allowAdditionalActions: true})
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
