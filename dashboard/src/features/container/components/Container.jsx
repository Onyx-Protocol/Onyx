import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import Loading from './Loading/Loading'

// Dashboard breaks if this isn't included at this level.
// TODO: investigate `actions` for circular dependencies?
import actions from 'actions'

class Container extends React.Component {
  render() {
    if (!this.props.loaded) {
      return(<Loading>Connecting to Chain Core...</Loading>)
    } else if (!this.props.authorized) {
      // this.props.showLogin()
      return(<div>adsf</div>)
    } else if (!this.props.configured) {
      return(<div>
        configure me
      </div>)
    }

    return(<div>
      good to go
    </div>)
  }
}

export default connect(
  (state) => ({
    loaded: state.core.loaded,
    configured: state.core.configured,
    authorized: state.core.authorized,
//     authOk: !state.core.requireClientToken || state.core.validToken,
//     configKnown: state.core.configKnown,
//     configured: state.core.configured,
//     onTestnet: state.core.onTestnet,
  }),
  (dispatch) => ({
    showLogin: () => dispatch(push('/login'))
//     fetchInfo: options => dispatch(actions.core.fetchCoreInfo(options)),
//     fetchTestnetInfo: () => dispatch(actions.testnet.fetchTestnetInfo()),
//     showRoot: () => dispatch(actions.app.showRoot),
//     showConfiguration: () => dispatch(actions.app.showConfiguration()),
  })
)(Container)
