import chain from '../chain'

import { useRouterHistory } from 'react-router'
import { createHistory } from 'history'

let apiHost = process.env.API_URL
let basename = "/"

if (process.env.NODE_ENV === 'production') {
  apiHost = window.location.origin
  basename = "/dashboard"
}

// API context
export const context = new chain.Context({
  url: apiHost
})

// react-router history object
export const history = useRouterHistory(createHistory)({
    basename: basename
})
