import generateListActions from './listActions'
import generateFormActions from './formActions'

const type = "mockhsm"

const list = generateListActions(type, {className: "MockHsm"})
const form = generateFormActions(type, {
  className: "MockHsm",
  resetAction: function(dispatch) {
    dispatch(list.resetPage())
  }
})

let actions = Object.assign({},
  list,
  form
)

export default actions
