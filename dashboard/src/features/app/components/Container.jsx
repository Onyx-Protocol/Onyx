import React from 'react'
import { connect } from 'react-redux'
import actions from 'actions'
import { Main, Config, Login, Modal } from './'

const CORE_POLLING_TIME=15000

class Container extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      loadedInfo: false
    }

    this.redirectRoot = this.redirectRoot.bind(this)
  }

  redirectRoot(configured, location) {
    if (configured || this.props.loginRequired) {
      if (location.pathname === '/' ||
          location.pathname.indexOf('configuration') >= 0) {
        this.props.showRoot()
      }
    } else {
      this.props.showConfiguration()
    }
  }

  componentWillMount() {
    this.props.fetchInfo().then(() => {
      this.setState({loadedInfo: true})
      this.redirectRoot(this.props.configured, this.props.location)
    })

    setInterval(this.props.fetchInfo, CORE_POLLING_TIME)
  }

  componentWillReceiveProps(nextProps) {
    if (nextProps.configured != this.props.configured ||
        nextProps.location.pathname != this.props.location.pathname) {
      this.redirectRoot(nextProps.configured, nextProps.location)
    }
  }

  render() {
    if (!this.state.loadedInfo) return(<div>Loading...</div>)

    let layout = <Main>{this.props.children}</Main>
    if (this.props.loginRequired && !this.props.loggedIn) {
      layout = <Login />
    } else if (!this.props.configured) {
      layout = <Config>{this.props.children}</Config>
    }

    return(<div>
      {layout}
      <Modal />
    </div>)
  }
}

export default connect(
  (state) => ({
    configured: state.core.configured,
    buildCommit: state.core.buildCommit,
    buildDate: state.core.buildDate,
    loginRequired: state.core.requireClientToken,
    loggedIn: state.core.validToken,
  }),
  (dispatch) => ({
    fetchInfo: options => dispatch(actions.core.fetchCoreInfo(options)),
    showRoot: () => dispatch(actions.routing.showRoot),
    showConfiguration: () => dispatch(actions.routing.showConfiguration()),
    clearSession: () => dispatch(actions.core.clearSession()),
  })
)(Container)
