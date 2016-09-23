import generateListActions from './listActions'
import generateFormActions from './formActions'

const type = 'asset'

const list = generateListActions(type, { defaultKey: 'alias' })
const form = generateFormActions(type, { jsonFields: ['tags', 'definition'] })

let actions = Object.assign({},
  list,
  form
)

export default actions
