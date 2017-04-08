import { baseCreateActions, baseUpdateActions, baseListActions } from 'features/shared/actions'

const type = 'asset'

const list = baseListActions(type, { defaultKey: 'alias' })
const create = baseCreateActions(type, {
  jsonFields: ['tags', 'definition'],
  intFields: ['quorum'],
  redirectToShow: true,
})
const update = baseUpdateActions(type, {
  jsonFields: ['tags']
})

const actions = {
  ...list,
  ...create,
  ...update,
}
export default actions
