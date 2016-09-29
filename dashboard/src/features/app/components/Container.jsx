import { connect } from 'react-redux'
import actions from '../../../actions'
import { Main, Config } from './'
import React from 'react'

const CORE_POLLING_TIME=15000

class Container extends React.Component {
  constructor(props) {
    super(props)

    this.state = { loadedInfo: false }

    this.redirectRoot = this.redirectRoot.bind(this)
  }

  redirectRoot(configured, location) {
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
    this.props.fetchInfo().then(() => {
      this.setState({loadedInfo: true})
      this.redirectRoot(this.props.configured, this.props.location)
    }).catch((err) => {
      this.setState({loadedInfo: true})
      throw(err)
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
    let loading = <div>Loading...</div>

    let layout = <Main>{this.props.children}</Main>
    if (!this.props.configured) {
      layout = <Config>{this.props.children}</Config>
    }

    return this.state.loadedInfo ? layout : loading
  }
}

export default connect(
  (state) => ({
    configured: state.core.configured,
    buildCommit: state.core.buildCommit,
    buildDate: state.core.buildDate
  }),
  (dispatch) => ({
    fetchInfo: () => dispatch(actions.core.fetchCoreInfo()),
    showRoot: () => dispatch(actions.routing.showRoot),
    showConfiguration: () => dispatch(actions.routing.showConfiguration()),
  })
)(Container)
