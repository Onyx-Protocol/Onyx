import React from 'react'
import { Route, IndexRoute, Redirect } from 'react-router'

import Navigation from './components/Navigation/Navigation'
import Home from './components/Home'
import NotFound from './components/NotFound'

export default ({
  path: '/',
  component: Navigation,
  indexRoute: { component: Home },
  childRoutes: [
    {
      path: '*',
      component: NotFound
    }
  ]
})
