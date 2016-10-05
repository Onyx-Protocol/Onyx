import uuid from 'uuid'
import chain from 'chain'
import { context } from 'utility/environment'
import { parseNonblankJSON } from 'utility/string'
import { push } from 'react-router-redux'

import generateListActions from 'actions/listActions'
import generateFormActions from 'actions/formActions'
import unspentActions from 'actions/unspent'

const type = 'transaction'

const list = generateListActions(type, {
  defaultKey: 'id'
})
const form = generateFormActions(type)

function preprocessTransaction(formParams) {
  const copy = JSON.parse(JSON.stringify(formParams))
  const builder = {
    base_transaction: copy.base_transaction,
    actions: copy.actions,
  }

  for (let i in builder.actions) {
    let a = builder.actions[i]
    try {
      a.reference_data = parseNonblankJSON(a.reference_data)
    } catch (err) {
      throw new Error(`Action ${parseInt(i)+1} reference data should be valid JSON, or blank.`)
    }
  }

  // HACK: Check for retire actions and replace with OP_FAIL control programs.
  // TODO: update JS SDK to support Java SDK builder style.
  for (let i = 0; i < builder.actions.length; i++) {
    let a = builder.actions[i]
    if (a.type == 'retire_asset') {
      a.type = 'control_program'
      a.control_program = '6a' // OP_FAIL hex byte
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
      .then(() => {
        dispatch(push('/transactions'))
        dispatch(form.created())
        dispatch(unspentActions.updateQuery(''))
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


let actions = Object.assign({},
  list,
  form
)

export default actions
