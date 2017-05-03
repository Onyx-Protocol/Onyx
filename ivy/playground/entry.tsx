import * as React from 'react';
import app from './app'
import { render } from 'react-dom';
import { applyMiddleware, compose, createStore } from 'redux'
import { Provider } from 'react-redux';
import { BrowserRouter as Router, Route } from 'react-router-dom'
import { ConnectedRouter, routerMiddleware } from 'react-router-redux'
import createHistory from 'history/createBrowserHistory'
import DocumentTitle from 'react-document-title'
import persistState from 'redux-localstorage'
import thunk from 'redux-thunk'
import { reset } from './app/actions'

import accounts from './accounts'
import assets from './assets'
import templates from './templates'

import Create from './contracts/components/create'
import Contracts from './contracts/components/contracts'
import Spend from './contracts/components/spend'
import { prefixRoute } from './util'

require('./static/playground.css')

interface ExtensionWindow extends Window {
  __REDUX_DEVTOOLS_EXTENSION_COMPOSE__: any
}
const composeEnhancers = (window as ExtensionWindow).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ || compose;

const history = createHistory()
const store = createStore(
  app.reducer,
  composeEnhancers(applyMiddleware(thunk), applyMiddleware(routerMiddleware(history)), persistState())
)

store.dispatch(reset)

render(
  <Provider store={store}>
    <DocumentTitle title='Ivy Playground'>
    <ConnectedRouter history={history}>
      <app.components.Root>
       <Route exact={true} path={prefixRoute('/')} component={templates.components.Editor} />
       <Route path={prefixRoute('/create')} component={Create} />
       <Route exact path={prefixRoute('/spend')}  component={Contracts} />
       <Route path={prefixRoute('/spend/:contractId')} component={Spend} />
      </app.components.Root>
    </ConnectedRouter>
    </DocumentTitle>
  </Provider>,
  document.getElementById('root')
)
