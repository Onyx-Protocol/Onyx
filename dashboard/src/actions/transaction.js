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

form.submitForm = (data) => function(dispatch) {
  // HACK: Check for retire actions and replace with OP_FAIL control programs.
  // TODO: update JS SDK to support Java SDK builder style.
  for (let i = 0; i < data.actions.length; i++) {
    let a = data.actions[i]
    if (a.type == 'retire_asset') {
      a.type = 'control_program'
      a.params.control_program = '6a' // OP_FAIL hex byte
    }
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
