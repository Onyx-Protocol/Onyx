import uuid from 'uuid'
import chain from '_chain'
import { context } from 'utility/environment'
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
    base_transaction: copy.base_transaction,
    actions: copy.actions,
  }

  if (builder.base_transaction == '') {
    delete builder.base_transaction
  }

  if (formParams.submit_action == 'generate') {
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
      a.control_program = '6a' // OP_FAIL hex byte
    }

    try {
      a.reference_data = parseNonblankJSON(a.reference_data)
    } catch (err) {
      throw new Error(`Action ${parseInt(i)+1} reference data should be valid JSON, or blank.`)
    }
  }

  return builder
}

function getTemplateXpubs(tpl) {
  const xpubs = []
  tpl.signing_instructions.forEach((instruction) => {
    instruction.witness_components.forEach((component) => {
      component.keys.forEach((key) => {
        xpubs.push(key.xpub)
      })
    })
  })
  return xpubs
}

form.submitForm = (formParams) => function(dispatch) {
  let builder
  try {
    builder = preprocessTransaction(formParams)
  } catch (err) {
    return Promise.reject(err)
  }

  const build = new chain.Transaction(builder).build(context())

  if (formParams.submit_action == 'submit') {
    return build
      .then(tpl => chain.MockHsm.sign([tpl], getTemplateXpubs(tpl), context()))
      .then(signed => signed[0].submit(context()))
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

  // submit_action == 'generate'
  return build
    .then(tpl => chain.MockHsm.sign(
      [{...tpl, allow_additional_actions: true}],
      getTemplateXpubs(tpl),
      context()
    ))
    .then(signed => {
      const id = uuid.v4()
      dispatch({
        type: 'GENERATED_TX_HEX',
        generated: {
          id: id,
          hex: signed[0].raw_transaction,
        },
      })
      dispatch(push(`/transactions/generated/${id}`))
    })
}

export default {
  ...list,
  ...form,
}
