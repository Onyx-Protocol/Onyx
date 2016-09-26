import chain from '../../chain'
import { context } from '../../utility/environment'

import generateListActions from '../../actions/listActions'
import generateFormActions from '../../actions/formActions'

const type = 'account'

const list = generateListActions(type, { defaultKey: 'alias' })
const form = generateFormActions(type, { jsonFields: ['tags'] })

let actions = Object.assign({},
  list,
  form,
  {
    createControlProgram: (data) => () =>
      chain.ControlProgram.create(data, context)
  }
)

export default actions
