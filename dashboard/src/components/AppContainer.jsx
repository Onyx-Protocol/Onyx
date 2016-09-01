import React from 'react'
import Main from '../containers/Layout/Main'
import Config from './Layout/Config'

const CORE_POLLING_TIME=15000

class AppContainer extends React.Component {
  constructor(props) {
    super(props)

    this.redirectRoot = this.redirectRoot.bind(this)
  }

  redirectRoot(configured, location) {
    if (configured) {
      if (location.pathname === "/" ||
          location.pathname.indexOf("configuration") >= 0) {
        this.props.showRoot()
      }
    } else {
      this.props.showConfiguration()
    }
  }

  componentWillMount() {
    this.props.fetchInfo().then(() => {
      this.redirectRoot(this.props.configured, this.props.location)
    })

    setInterval(this.props.fetchInfo, CORE_POLLING_TIME)
  }

  componentWillReceiveProps(nextProps) {
    if (nextProps.configured != this.props.configured &&
        nextProps.location.pathName != this.props.location.pathName) {
      this.redirectRoot(nextProps.configured, nextProps.location)
    }
  }

  render() {
    let layout = <Main>{this.props.children}</Main>
    if (!this.props.configured) {
      layout = <Config>{this.props.children}</Config>
    }

    return layout
  }
}

export default AppContainer
