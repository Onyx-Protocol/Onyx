import chain from '../chain'
import { context } from '../utility/environment'
import { push } from 'react-router-redux'

import generateListActions from './listActions'
import generateFormActions from './formActions'
import unspentActions from './unspent'

const type = "transaction"

const list = generateListActions(type, {
  defaultKey: "id"
})
const form = generateFormActions(type)

function parseNonblankJSON(json) {
  json = json || ''
  json = json.trim()

  if (json == '') {
    return null
  }

  return JSON.parse(json)
}

function preprocessTransaction(data) {
  try {
    data.reference_data = parseNonblankJSON(data.reference_data)
  } catch (err) {
    throw new Error("Transaction-level reference data should be valid JSON, or blank.")
  }

  for (let i in data.actions) {
    let a = data.actions[i]
    try {
      a.reference_data = parseNonblankJSON(a.reference_data)
    } catch (err) {
      throw new Error(`Action ${parseInt(i)+1} reference data should be valid JSON, or blank.`)
    }
  }

  // HACK: Check for retire actions and replace with OP_FAIL control programs.
  // TODO: update JS SDK to support Java SDK builder style.
  for (let i = 0; i < data.actions.length; i++) {
    let a = data.actions[i]
    if (a.type == 'retire_asset') {
      a.type = 'control_program'
      a.params.control_program = '6a' // OP_FAIL hex byte
    }
  }
}

form.submitForm = (data) => function(dispatch) {
  try {
    preprocessTransaction(data)
  } catch (err) {
    return Promise.reject(err)
  }

  return new chain.Transaction(data)
    .build(context)
    .then((template) => {
      return chain.MockHsm.sign([template], context)
    })
    .then((signedTemplates) => {
      return signedTemplates[0].submit(context)
    })
    .then(() => {
      dispatch(list.updateQuery(""))
      dispatch(list.resetPage())
      dispatch(unspentActions.resetPage())
      dispatch(push('/transactions'))
    })
}


let actions = Object.assign({},
  list,
  form
)

export default actions
