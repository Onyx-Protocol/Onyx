import * as chain from 'chain-sdk'

const client = new chain.Client()
const signer = new chain.HsmSigner()

export { client, signer }
