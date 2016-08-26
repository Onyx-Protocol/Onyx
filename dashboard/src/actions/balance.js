import generateListActions from './listActions'

const type = "balance"

const list = generateListActions(type, {
  defaultSumBy: () => ["asset_id","asset_alias"]
})

let actions = Object.assign({},
  list
)

export default actions
