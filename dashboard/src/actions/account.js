import generateListActions from './listActions'
import generateFormActions from './formActions'

const type = "account"

const list = generateListActions(type, { defaultKey: "alias" })
const form = generateFormActions(type, {
  resetAction: function(dispatch) {
    dispatch(list.updateQuery(""))
    dispatch(list.resetPage())
  }
})

let actions = Object.assign({},
  list,
  form
)

export default actions
