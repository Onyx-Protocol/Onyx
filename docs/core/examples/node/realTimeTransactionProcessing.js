const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key, feed

// snippet processor-method
processTransaction = (tx) => {
  console.log(`New transaction at ${tx.timestamp}`)
  console.log(`\tID: ${tx.id}`)

  tx.inputs.forEach((input, index) => {
    console.log(`\tInput ${index}`)
    console.log(`\t\tType: ${input.type}`)
    console.log(`\t\tAsset: ${input.asset_alias}`)
    console.log(`\t\tAmount: ${input.amount}`)
    console.log(`\t\tAccount: ${input.account_alias}`)
  })

  tx.outputs.forEach((output, index) => {
    console.log(`\tOutput ${index}`)
    console.log(`\t\tType: ${output.type}`)
    console.log(`\t\tPurpose: ${output.purpose}`)
    console.log(`\t\tAsset: ${output.asset_alias}`)
    console.log(`\t\tAmount: ${output.amount}`)
    console.log(`\t\tAccount: ${output.account_alias}`)
  })
}
// endsnippet

client.mockHsm.keys.create()
.then(_key => {
  key = _key.xpub
  signer.addKey(key, client.mockHsm.signerUrl)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    root_xpubs: [key],
    quorum: 1,
  }),
  client.accounts.create({
    alias: 'alice',
    root_xpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    root_xpubs: [key],
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
  client.transactions.build( function(builder){
    builder.issue({
      asset_alias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 100
    })
  })
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
  // endsnippet
).then(() =>
  // snippet transfer
  client.transactions.build( function(builder){
    builder.spendFromAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 50
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 50
    })
  })
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
  // endsnippet
)
