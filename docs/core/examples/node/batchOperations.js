const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

client.mockHsm.keys.create()
.then(Key => {
  key = Key
  signer.addKey(key.xpub, client.mockHsm.signerConnection)
}).then(() => {
  // snippet asset-builders
  const assetsToBuild = [{
    alias: 'gold',
    rootXpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'silver',
    rootXpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'bronze',
    rootXpubs: [key.xpub],
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
}).catch(err =>
  process.nextTick(() => { throw err })
)
