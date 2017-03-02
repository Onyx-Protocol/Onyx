const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

client.mockHsm.keys.create().then(_key => {
  key = _key
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
}).then(() => {
  // snippet nondeterministic-errors
  const assetsToBuild = [{
    alias: 'platinum',
    rootXpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'platinum',
    rootXpubs: [key.xpub],
    quorum: 1,
  }, {
    alias: 'platinum',
    rootXpubs: [key.xpub],
    quorum: 1,
  }]
  // endsnippet

  return client.assets.createBatch(assetsToBuild)
}).then(assetBatch => {
  assetBatch.errors.forEach((err, index) => {
    console.log(`asset ${index} error: `)
    console.log(err)
  })

  assetBatch.successes.forEach((asset, index) => {
    console.log(`asset ${index} created, ID: ${asset.id}`)
  })
}).then(() => client.accounts.createBatch([
  { alias: 'alice', rootXpubs: [key.xpub], quorum: 1 },
  { alias: 'bob', rootXpubs: [key.xpub], quorum: 1 },
])).then(() => {
  // snippet batch-build-builders
  const transactionsToBuild = [
    (builder) => {
      builder.issue({assetAlias: 'gold', amount: 100})
      builder.controlWithAccount({accountAlias: 'alice', assetAlias: 'gold', amount: 100})
    },
    (builder) => {
      builder.issue({assetAlias: 'not-a-real-asset', amount: 100})
      builder.controlWithAccount({accountAlias: 'alice', assetAlias: 'not-a-real-asset', amount: 100})
    },
    (builder) => {
      builder.issue({assetAlias: 'silver', amount: 100})
      builder.controlWithAccount({accountAlias: 'alice', assetAlias: 'silver', amount: 100})
    }
  ]
  // endsnippet

  // snippet batch-build-handle-errors
  const buildBatchPromise = client.transactions.buildBatch(transactionsToBuild)
    .then((buildBatch) => {
      buildBatch.errors.forEach((err, index) => {
        console.log(`Error building transaction ${index} error: `)
        console.log(err)
      })

      return buildBatch
    })
  // endsnippet

  return buildBatchPromise
}).then((buildBatch) => {
  // snippet batch-sign
  const transactionsToSign = buildBatch.successes
  const signBatchPromise = signer.signBatch(transactionsToSign)
    .then((signBatch) => {
      signBatch.errors.forEach((err, index) => {
        console.log(`Error signing transaction ${index} error: `)
        console.log(err)
      })

      return signBatch
    })
  // endsnippet

  return signBatchPromise
}).then((signBatch) => {
  // snippet batch-submit
  const transactionsToSubmit = signBatch.successes
  const submitBatchPromise = client.transactions.submitBatch(transactionsToSubmit)
    .then((submitBatch) => {
      submitBatch.errors.forEach((err, index) => {
        console.log(`Error submitting transaction ${index} error: `)
        console.log(err)
      })

      submitBatch.successes.forEach((tx, index) => {
        console.log(`Transaction ${index} submitted, ID: ${tx.id}`)
      })

      return submitBatch
    })
  // endsnippet
}).catch(err =>
  process.nextTick(() => { throw err })
)
