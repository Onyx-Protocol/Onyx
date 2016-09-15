import chain from '../chain'
import { context } from '../utility/environment'

import generateListActions from './listActions'
import generateFormActions from './formActions'

const type = "account"

const list = generateListActions(type, { defaultKey: "alias" })
const form = generateFormActions(type)

let actions = Object.assign({},
  list,
  form,
  {
    createControlProgram: (data) => () =>
      chain.ControlProgram.create(data, context)
  }
)

export default actions
