import { connect } from 'react-redux'
import actions from '../actions'
import AppContainer from '../components/AppContainer'

const mapStateToProps = (state) => ({
  configured: state.core.configured,
  buildCommit: state.core.buildCommit,
  buildDate: state.core.buildDate
})

const mapDispatchToProps = (dispatch) => ({
  fetchInfo: () => dispatch(actions.core.fetchCoreInfo()),
  showRoot: () => dispatch(actions.routing.showRoot),
  showConfiguration: () => dispatch(actions.routing.showConfiguration()),

  // FIXME: seeding keys should not trigger anything related to paging
  prefetchKeys: () => dispatch(actions.mockhsm.displayNextPage())
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(AppContainer)
