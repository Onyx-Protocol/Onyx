import { createStore, applyMiddleware } from 'redux'
import thunkMiddleware from 'redux-thunk'
import createLogger from 'redux-logger'
import { routerMiddleware as createRouterMiddleware } from 'react-router-redux'
import { history } from './utility/environment'

import rootReducer from './reducers'

const loggerMiddleware = createLogger()
const routerMiddleware = createRouterMiddleware(history)

export default function() {
  const store = createStore(
  	rootReducer,
  	applyMiddleware(
      thunkMiddleware,
      loggerMiddleware,
      routerMiddleware
    )
  )

  if (module.hot) {
    // Enable Webpack hot module replacement for reducers
    module.hot.accept('./reducers', () => {
      const nextRootReducer = require('./reducers/index');
      store.replaceReducer(nextRootReducer);
    });
  }

  return store;
}
