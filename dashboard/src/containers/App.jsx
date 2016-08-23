import React from 'react'
import { Provider } from 'react-redux'
import { Router, Route, IndexRoute } from 'react-router'
import { history } from '../utility/environment'
import { syncHistoryWithStore } from 'react-router-redux'

import routes from '../routes'

const basename = process.env.NODE_ENV === "production" ? "/dashboard" : "/"

export default class App extends React.Component {
  componentWillMount() {
    document.title = "Chain Core Dashboard"
  }

  render() {
    const store = this.props.store
    const syncedHistory = syncHistoryWithStore(history, store)
    return (
      <Provider store={store}>
        <Router history={syncedHistory} routes={routes} />
      </Provider>
    )
  }
}
