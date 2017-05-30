const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

// snippet processor-method
const processTransaction = tx => {
  console.log(`New transaction at ${tx.timestamp}`)
  console.log(`\tID: ${tx.id}`)

  tx.inputs.forEach((input, index) => {
    console.log(`\tInput ${index}`)
    console.log(`\t\tType: ${input.type}`)
    console.log(`\t\tAsset: ${input.assetAlias}`)
    console.log(`\t\tAmount: ${input.amount}`)
    console.log(`\t\tAccount: ${input.accountAlias}`)
  })

  tx.outputs.forEach((output, index) => {
    console.log(`\tOutput ${index}`)
    console.log(`\t\tType: ${output.type}`)
    console.log(`\t\tPurpose: ${output.purpose}`)
    console.log(`\t\tAsset: ${output.assetAlias}`)
    console.log(`\t\tAmount: ${output.amount}`)
    console.log(`\t\tAccount: ${output.accountAlias}`)
  })
}
// endsnippet

client.mockHsm.keys.create()
.then(Key => {
  key = Key.xpub
  signer.addKey(key, client.mockHsm.signerConnection)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    rootXpubs: [key],
    quorum: 1,
  }),
  client.accounts.create({
    alias: 'alice',
    rootXpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    rootXpubs: [key],
    quorum: 1
  }),

  // snippet create-feed
  client.transactionFeeds.create({
    alias: 'local-transactions',
    filter: "is_local='yes'"
  })
  // endsnippet
]))
.then(() => {

  // snippet get-feed
  const feedPromise = client.transactionFeeds.get({
    alias: 'local-transactions'
  })
  // endsnippet

  return feedPromise
})
.then(feed => {

  // snippet processing-loop
  const processingLoop = (tx, next, done, fail) => {
    processTransaction(tx)
    next(true)
  }
  // endsnippet

  // snippet processing-thread
  // JavaScript is single-threaded, and uses a callback for processing
  feed.consume(processingLoop)
  // endsnippet
})
.then(() =>
  // snippet issue
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 100
    })
  })
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
  // endsnippet
).then(() =>
  // snippet transfer
  client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 50
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 50
    })
  })
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
  // endsnippet
).catch(err =>
  process.nextTick(() => { throw err })
)
