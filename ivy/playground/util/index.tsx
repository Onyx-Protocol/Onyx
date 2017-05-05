const chain = require('chain-sdk')

let isProd: boolean = process.env.NODE_ENV === 'production'
let apiHost: string
if (isProd) {
  apiHost = window.location.origin
} else {
  apiHost = process.env.API_URL || 'http://localhost:8080/api'
}

export const client = new chain.Client({
  url: apiHost
})

export const signer = new chain.HsmSigner()

export const prefixRoute = (route: string): string => {
  if (isProd) {
    return "/ivy" + route
  }
  return route
}
