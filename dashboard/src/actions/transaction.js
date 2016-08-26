import chain from '../chain'
import { context } from '../utility/environment'
import { push } from 'react-router-redux'

import generateListActions from './listActions'
import generateFormActions from './formActions'
import unspentActions from './unspent'

const type = "transaction"

const list = generateListActions(type, { tryId: true })
const form = generateFormActions(type)

form.submitForm = (data) => function(dispatch) {
  let transaction = new chain.Transaction(data).build(context)

  transaction.then((template) => {
    return chain.MockHsm.sign([template], context)
  })
  .then((signedTemplates) => {
    return chain.Transaction.submit(signedTemplates, context)
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
