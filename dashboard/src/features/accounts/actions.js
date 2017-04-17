import { chainClient } from 'utility/environment'
import { baseCreateActions, baseUpdateActions, baseListActions } from 'features/shared/actions'

const type = 'account'

const list = baseListActions(type, { defaultKey: 'alias' })
const create = baseCreateActions(type, {
  jsonFields: ['tags'],
  intFields: ['quorum'],
  redirectToShow: true,
})
const update = baseUpdateActions(type, {
  jsonFields: ['tags']
})

let actions = {
  ...list,
  ...create,
  ...update,
  createReceiver: (data) => () => {
    return chainClient().accounts.createReceiver(data)
  }
}

export default actions
