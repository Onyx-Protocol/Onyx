import React from 'react'
import { connect } from 'react-redux'
import Loading from './Loading/Loading'
import { showLogin, showConfiguration, showRoot } from 'features/bootstrap/actions'

// Dashboard breaks if this isn't included at this level.
// TODO: investigate `actions` for circular dependencies?
import actions from 'actions'

class Container extends React.Component {

  componentWillReceiveProps(nextProps) {
    if (!nextProps.authenticationReady) return

    const pathname = nextProps.location.pathname
    if (nextProps.shouldShowLogin) {
      if (pathname != '/login') {
        this.props.showLogin()
      }
    } else if (nextProps.shouldShowConfig) {
      if (pathname != '/configuration') {
        this.props.showConfiguration()
      }
    } else if (['/', '/login', '/configuration'].includes(pathname)) {
      this.props.showRoot()
    }
  }

  render() {
    if (!this.props.authenticationReady) {
      return(<Loading>Connecting to Chain Core...</Loading>)
    }

    return this.props.children
  }
}

export default connect(
  (state) => ({
    authenticationReady: state.authn.authenticationReady,
    shouldShowConfig: !state.core.configured,
    shouldShowLogin: state.authn.authenticationRequired && !state.authn.authenticated,
  }),
  (dispatch) => ({
    showLogin: () => dispatch(showLogin()),
    showRoot: () => dispatch(showRoot()),
    showConfiguration: () => dispatch(showConfiguration()),
  })
)(Container)
