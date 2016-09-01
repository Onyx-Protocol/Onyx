import { connect } from 'react-redux'
import actions from '../../actions'
import MainComponent from '../../components/Layout/Main'

const mapStateToProps = (state) => ({
  dropdownState: state.app.dropdownState,
  buildCommit: state.core.buildCommit,
  buildDate: state.core.buildDate
})

const mapDispatchToProps = (dispatch) => ({
  toggleDropdown: () => dispatch(actions.app.toggleDropdown()),
  closeDropdown: () => dispatch(actions.app.closeDropdown())
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(MainComponent)
