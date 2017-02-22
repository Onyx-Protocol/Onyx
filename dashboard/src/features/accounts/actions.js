import { chainClient } from 'utility/environment'
import { baseCreateActions, baseListActions } from 'features/shared/actions'

const type = 'account'

const list = baseListActions(type, { defaultKey: 'alias' })
const form = baseCreateActions(type, {
  jsonFields: ['tags'],
  intFields: ['quorum'],
  redirectToShow: true,
})

let actions = {
  ...list,
  ...form,
  createControlProgram: (data) => () => {
    return chainClient().accounts.createControlProgram(data)
  }
}

export default actions
