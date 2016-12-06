const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key, feed

processTransaction = (tx) => {
  console.log('New transaction at ' + tx.timestamp)
  console.log('ID: ' + tx.id)

  tx.inputs.forEach((input, index) => {
    console.log('Input ' + index)
    console.log('Type: ' + input.type)
    console.log('Asset: ' + input.asset_alias)
    console.log('Amount: ' + input.amount)
    console.log('Account: ' + input.account_alias)
  })

  tx.outputs.forEach((output, index) => {
    console.log('Output ' + index)
    console.log('Type: ' + output.type)
    console.log('Purpose: ' + output.purpose)
    console.log('Asset: ' + output.asset_alias)
    console.log('Amount: ' + output.amount)
    console.log('Account: ' + output.account_alias)
  })
}

Promise.all([
  client.transactionFeeds.get({alias: 'local-transactions'})
]).then(feeds => {
  feed = feeds[0]
}).then(() => feed.consume())

// client.transactionFeeds.get({alias: 'local-transactions'})
//   .then((data) => console.log(data))
//
// Promise.all([
//   client.mockHsm.keys.create(),
// ]).then(keys => {
//   key  = keys[0].xpub
//
//   signer.addKey(key, client.mockHsm.signerUrl)
// }).then(() => Promise.all([
//
//   client.assets.create({
//     alias: 'gold',
//     root_xpubs: [key],
//     quorum: 1,
//   }),
//
//   client.accounts.create({
//     alias: 'alice',
//     root_xpubs: [key],
//     quorum: 1
//   }),
//
//   client.accounts.create({
//     alias: 'bob',
//     root_xpubs: [key],
//     quorum: 1
//   }),
//
//   // snippet create-feed
//   client.transactionFeeds.create({
//     alias: 'local-transactions',
//     filter: "is_local='yes'"
//   })
//   // endsnippet
// ])
// ).then(() => Promise.all([
//   client.transactionFeeds.get({
//     alias: 'local-transactions'
//   })
// ]))
//   .then(feeds => {
//     feed = feeds[0]
// }).then(() =>
//   client.transactions.build( function(builder){
//     builder.issue({
//       asset_alias: 'gold',
//       amount: 100
//     })
//     builder.controlWithAccount({
//       account_alias: 'alice',
//       asset_alias: 'gold',
//       amount: 100
//     })
//   })
//   .then((issuance) => signer.sign(issuance))
//   .then((signed) => client.transactions.submit(signed))
// ).then(() =>
//   client.transactions.build( function(builder){
//     builder.spendFromAccount({
//       account_alias: 'alice',
//       asset_alias: 'gold',
//       amount: 50
//     })
//     builder.controlWithAccount({
//       account_alias: 'bob',
//       asset_alias: 'gold',
//       amount: 50
//     })
//   }).then((transfer) => signer.sign(transfer))
//     .then((signed) => client.transactions.submit(signed))
// )
