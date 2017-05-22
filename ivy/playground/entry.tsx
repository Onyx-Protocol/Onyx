// external imports
import * as React from 'react'
import { render } from 'react-dom'
import { applyMiddleware, compose, createStore } from 'redux'
import { Provider } from 'react-redux';
import { BrowserRouter as Router, Route } from 'react-router-dom'
import { ConnectedRouter, routerMiddleware } from 'react-router-redux'
import createHistory from 'history/createBrowserHistory'
import DocumentTitle from 'react-document-title'
import persistState from 'redux-localstorage'
import thunk from 'redux-thunk'

// ivy imports
import app from './app'
import accounts from './accounts'
import assets from './assets'
import templates from './templates'
import { prefixRoute } from './core'

// TODO(boymanjor): Handle these imports in a better way
import LockedValue from './contracts/components/lockedValue'
import Lock from './templates/components/lock'
import Unlock from './contracts/components/unlock'

// Import css
require('./static/playground.css')

// Set favicon
const faviconPath = require('!!url?name=favicon.ico!./static/images/favicon.png')
const favicon = document.createElement('link')
favicon.type = 'image/png'
favicon.rel = 'shortcut icon'
favicon.href = faviconPath
document.getElementsByTagName('head')[0].appendChild(favicon)

interface ExtensionWindow extends Window {
  __REDUX_DEVTOOLS_EXTENSION_COMPOSE__: any
}
const composeEnhancers = (window as ExtensionWindow).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ || compose


const history = createHistory()
const store = createStore(
  app.reducer,
  composeEnhancers(applyMiddleware(thunk), applyMiddleware(routerMiddleware(history)), persistState())
)

const selected = templates.selectors.getSelectedTemplate(store.getState())
store.dispatch(templates.actions.loadTemplate(selected))

store.dispatch(app.actions.seed())

render(
  <Provider store={store}>
    <DocumentTitle title='Ivy Playground'>
    <ConnectedRouter history={history}>
      <app.components.Root>
       <Route exact={true} path={prefixRoute('/')} component={Lock} />
       <Route exact path={prefixRoute('/unlock')}  component={LockedValue} />
       <Route path={prefixRoute('/unlock/:contractId')} component={Unlock} />
      </app.components.Root>
    </ConnectedRouter>
    </DocumentTitle>
  </Provider>,
  document.getElementById('root')
)
