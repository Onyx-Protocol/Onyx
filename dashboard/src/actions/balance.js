import generateListActions from './listActions'

const type = "balance"

const list = generateListActions(type, {
  defaultSumBy: () => ["asset_id"]
})

let actions = Object.assign({},
  list
)

export default actions
