import React from 'react'
import { Provider } from 'react-redux'

import { Router, Route, IndexRoute, browserHistory } from 'react-router'
import { syncHistoryWithStore } from 'react-router-redux'

import routes from '../routes'


export default class App extends React.Component {
  componentWillMount() {
    document.title = "🚧 Chain Dashboard 🚧"
  }

  render() {
    const store = this.props.store
    const history = syncHistoryWithStore(browserHistory, store)

    return (
      <Provider store={store}>
        <Router history={history} routes={routes} />
      </Provider>
    )
  }
}
