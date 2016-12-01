const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

client.mockHsm.keys.create()
.then(_key => {
  key = _key
  signer.addKey(key.xpub, client.mockHsm.signerUrl)
}).then(() => {
  // snippet asset-builders
  const assetsToBuild = [{
    alias: 'gold',
    root_xpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'silver',
    root_xpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'bronze',
    root_xpubs: [key.xpub],
    quorum: 0,
  }]
  // endsnippet

  // snippet asset-create-batch
  const assetBatchPromise = client.assets.createBatch(assetsToBuild)
  // endsnippet

  return assetBatchPromise
}).then(assetBatch => {
  // snippet asset-create-handle-errors
  assetBatch.errors.forEach((err, index) => {
    console.log(`asset ${index} error: `)
    console.log(err)
  })

  assetBatch.successes.forEach((asset, index) => {
    console.log(`asset ${index} created, ID: ${asset.id}`)
  })
  // endsnippet
})
