import generateListActions from './listActions'

const type = "unspent"

const list = generateListActions(type)

let actions = Object.assign({},
  list
)

export default actions
