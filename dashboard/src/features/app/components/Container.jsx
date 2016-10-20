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

  redirectRoot(authOk, configured, location) {
    if (!authOk) {
      return
    }

    if (configured) {
      if (location.pathname === '/' ||
          location.pathname.indexOf('configuration') >= 0) {
        this.props.showRoot()
      }
    } else {
      this.props.showConfiguration()
    }
  }

  componentWillMount() {
    const checkInfo = () => {
      if (this.props.onTestNet) this.props.fetchTestNetInfo()
      return this.props.fetchInfo()
    }

    checkInfo().then(() => {
      this.setState({loadedInfo: true})
      this.redirectRoot(this.props.authOk, this.props.configured, this.props.location)
    })

    setInterval(checkInfo, CORE_POLLING_TIME)
  }

  componentWillReceiveProps(nextProps) {
    if (nextProps.authOk != this.props.authOk ||
        nextProps.configured != this.props.configured ||
        nextProps.location.pathname != this.props.location.pathname) {
      this.redirectRoot(nextProps.authOk, nextProps.configured, nextProps.location)
    }
  }

  render() {
    if (!this.state.loadedInfo) return(<div>Loading...</div>)

    let layout = <Main>{this.props.children}</Main>
    if (!this.props.authOk) {
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
    authOk: !state.core.requireClientToken || state.core.validToken,
    onTestNet: state.core.onTestNet,
  }),
  (dispatch) => ({
    fetchInfo: options => dispatch(actions.core.fetchCoreInfo(options)),
    fetchTestNetInfo: () => dispatch(actions.configuration.fetchTestNetInfo()),
    showRoot: () => dispatch(actions.app.showRoot),
    showConfiguration: () => dispatch(actions.app.showConfiguration()),
    clearSession: () => dispatch(actions.core.clearSession()),
  })
)(Container)
