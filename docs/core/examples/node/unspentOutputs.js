const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key, issuanceTxId

client.mockHsm.keys.create()
.then(_key => {
  key = _key
  signer.addKey(key.xpub, client.mockHsm.signerUrl)
})
.then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    root_xpubs: [key.xpub],
    quorum: 1,
  }),
  client.accounts.create({
    alias: 'alice',
    root_xpubs: [key.xpub],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    root_xpubs: [key.xpub],
    quorum: 1
  })
]))
.then(() =>
  client.transactions.build( function(builder){
    builder.issue({
      asset_alias: 'gold',
      amount: 200
    })
    builder.controlWithAccount({
      account_alias: 'alice',
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
)
.then(issuanceTx => {
  issuanceTxId = issuanceTx.id
})
.then(() =>
  // snippet alice-unspent-outputs
  client.unspentOutputs.query({
    filter: 'account_alias=$1',
    filter_params: ['alice'],
  }).then(aliceUnspentOutputs => {
    aliceUnspentOutputs.items.forEach((utxo) => {
      console.log(`Unspent output in alice account: ${utxo.transaction_id}:${utxo.position}`)
    })
  })
  // endsnippet
)
.then(() =>
  // snippet gold-unspent-outputs
  client.unspentOutputs.query({
    filter: 'asset_alias=$1',
    filter_params: ['gold'],
  }).then(goldUnspentOutputs => {
    goldUnspentOutputs.items.forEach((utxo) => {
      console.log(`Unspent output containing gold: ${utxo.transaction_id}:${utxo.position}`)
    })
  })
  // endsnippet
)
.then(() =>
  // snipped build-transaction-all
  client.transactions.build( function(builder){
    builder.spendUnspentOutput({
      transaction_id: issuanceTxId,
      position: 0,
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 100
    })
  })
  // endsnippet
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
)
.then(() =>
  // snippet build-transaction-partial
  client.transactions.build( function(builder){
    builder.spendUnspentOutput({
      transaction_id: issuanceTxId,
      position: 1,
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 40
    })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 60
    })
  })
  // endsnippet
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
)
