/* global process */

import chain from '../chain'
import { store } from 'app'

import { useRouterHistory } from 'react-router'
import { createHistory } from 'history'

let apiHost, basename
if (process.env.NODE_ENV === 'production') {
  apiHost = window.location.origin
  basename = '/dashboard'
} else {
  apiHost = process.env.API_URL || 'http://localhost:3000/api'
  basename = '/'
}

// API context
export const context = () => new chain.Context({
  url: apiHost,
  clientToken: store.getState().core.clientToken
})

// react-router history object
export const history = useRouterHistory(createHistory)({
  basename: basename
})

export const pageSize = 25
