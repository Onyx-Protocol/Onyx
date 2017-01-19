import { connect as reduxConnect } from 'react-redux'
import actions from 'actions'

export const mapStateToProps = ( type ) => ( /* state */ ) => ({
  type: type,
})

export const mapDispatchToProps = (type) => (dispatch) => ({
  submitForm: (data) => {
    dispatch(actions.tutorial.updateTutorial(data, type))
    return dispatch(actions[type].submitForm(data))
  }
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
