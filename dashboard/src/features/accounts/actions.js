import chain from 'chain'
import { context } from 'utility/environment'
import { baseFormActions, baseListActions } from 'features/shared/actions'

const type = 'account'

const list = baseListActions(type, { defaultKey: 'alias' })
const form = baseFormActions(type, {
  jsonFields: ['tags'],
  intFields: ['quorum'],
})

let actions = {
  ...list,
  ...form,
  createControlProgram: (data) => () =>
    chain.ControlProgram.create(data, context())
}

export default actions
