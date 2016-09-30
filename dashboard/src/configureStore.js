/*eslint-env node*/

import { createStore, applyMiddleware, compose } from 'redux'
import thunkMiddleware from 'redux-thunk'
import { routerMiddleware as createRouterMiddleware } from 'react-router-redux'
import { history } from './utility/environment'

import makeRootReducer from './reducers'

const routerMiddleware = createRouterMiddleware(history)

export default function() {
  const store = createStore(
    makeRootReducer(),
    compose(
      applyMiddleware(
        thunkMiddleware,
        routerMiddleware
      ),
      window.devToolsExtension ? window.devToolsExtension() : f => f
    )
  )

  if (module.hot) {
    // Enable Webpack hot module replacement for reducers
    module.hot.accept('./reducers', () => {
      const reducers = require('./reducers').default
      store.replaceReducer(reducers())
    })
  }

  return store
}
