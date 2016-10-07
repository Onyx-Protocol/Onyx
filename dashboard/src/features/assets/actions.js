import { baseFormActions, baseListActions } from 'features/shared/actions'

const type = 'asset'

const list = baseListActions(type, { defaultKey: 'alias' })
const form = baseFormActions(type, { jsonFields: ['tags', 'definition'] })

const actions = {
  ...list,
  ...form,
}
export default actions
