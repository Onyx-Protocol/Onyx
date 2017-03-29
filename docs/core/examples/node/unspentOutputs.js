const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key, issuanceTxId

client.mockHsm.keys.create()
.then(Key => {
  key = Key
  signer.addKey(key.xpub, client.mockHsm.signerConnection)
})
.then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    rootXpubs: [key.xpub],
    quorum: 1,
  }),
  client.accounts.create({
    alias: 'alice',
    rootXpubs: [key.xpub],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    rootXpubs: [key.xpub],
    quorum: 1
  })
]))
.then(() =>
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'gold',
      amount: 200
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
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
)
.then(issuanceTx => {
  issuanceTxId = issuanceTx.id
})
.then(() =>
  // snippet alice-unspent-outputs
  client.unspentOutputs.queryAll({
    filter: 'account_alias=$1',
    filterParams: ['alice'],
  }, (utxo, next) => {
    console.log(`Unspent output in alice account: ${utxo.transactionId}:${utxo.position}`)
    next()
  })
  // endsnippet
)
.then(() =>
  // snippet gold-unspent-outputs
  client.unspentOutputs.queryAll({
    filter: 'asset_alias=$1',
    filterParams: ['gold'],
  }, (utxo, next) => {
    console.log(`Unspent output containing gold: ${utxo.transactionId}:${utxo.position}`)
    next()
  })
  // endsnippet
)
.then(() =>
  // snipped build-transaction-all
  client.transactions.build(builder => {
    builder.spendUnspentOutput({
      transactionId: issuanceTxId,
      position: 0,
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 100
    })
  })
  // endsnippet
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
)
.then(() =>
  // snippet build-transaction-partial
  client.transactions.build(builder => {
    builder.spendUnspentOutput({
      transactionId: issuanceTxId,
      position: 1,
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 40
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 60
    })
  })
  // endsnippet
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
).catch(err =>
  process.nextTick(() => { throw err })
)
