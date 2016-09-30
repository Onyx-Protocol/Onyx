import { connect as reduxConnect } from 'react-redux'
import actions from '../../../actions'

export const mapStateToProps = ( type ) => ( /* state */ ) => ({
  type: type
})

export const mapDispatchToProps = (type) => (dispatch) => {
  return {
    submitForm: (data) => dispatch(actions[type].submitForm(data))
  }
}

export const connect = (state, dispatch, element) => reduxConnect(
  state,
  dispatch
)(element)

export default {
  mapStateToProps,
  mapDispatchToProps,
  connect,
}
