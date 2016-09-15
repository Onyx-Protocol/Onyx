import generateListActions from './listActions'
import generateFormActions from './formActions'

const type = "index"

const list = generateListActions(type)
const form = generateFormActions(type, {
  listPath: "indexes",
  createPath: "indexes/create"
})

let actions = Object.assign({},
  list,
  form
)

export default actions
