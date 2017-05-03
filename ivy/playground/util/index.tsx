import * as chain from 'chain-sdk'

let apiHost
if (process.env.NODE_ENV === 'production') {
  apiHost = window.location.origin
} else {
  apiHost = process.env.API_URL || 'http://localhost:8080/api'
}

export const client = new chain.Client({
  baseUrl: apiHost
})

export const signer = new chain.HsmSigner()
