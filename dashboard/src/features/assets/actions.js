import generateListActions from '../../actions/listActions'
import generateFormActions from '../../actions/formActions'

const type = 'asset'

const list = generateListActions(type, { defaultKey: 'alias' })
const form = generateFormActions(type, { jsonFields: ['tags', 'definition'] })

const actions = {
  ...list,
  ...form,
}
export default actions
