import generateListActions from './listActions'

const type = "balance"

const list = generateListActions(type)

let actions = Object.assign({},
  list
)

export default actions
