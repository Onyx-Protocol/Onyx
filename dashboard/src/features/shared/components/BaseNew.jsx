import { connect as reduxConnect } from 'react-redux'
import actions from 'actions'

export const mapStateToProps = ( type ) => ( /* state */ ) => ({
  type: type,
})

export const mapDispatchToProps = (type) => (dispatch) => ({
  submitForm: (data) => dispatch(actions[type].submitForm(data)),
  updateTutorial: (data) => dispatch(actions.tutorial.updateTutorial(data, type))
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
