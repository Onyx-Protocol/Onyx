import { connect as reduxConnect } from 'react-redux'
import actions from 'actions'

export const mapStateToProps = ( type ) => ( /* state */ ) => ({
  type: type,
})

export const mapDispatchToProps = (type) => (dispatch) => ({
  fetchItem: (id) => dispatch(actions[type].fetchItems({filter: `id='${id}'`})).then((resp) => {
    return resp
  }),
  submitForm: (data, id) => dispatch(actions[type].submitUpdateForm(data, id))
})

export const connect = (state, dispatch, element) => reduxConnect(
  state,
  dispatch
)(element)

export default {
  mapStateToProps,
  mapDispatchToProps,
  connect,
}
