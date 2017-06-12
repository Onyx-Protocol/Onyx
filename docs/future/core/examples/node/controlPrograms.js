const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()

client.mockHsm.keys.create().then(key => {
  signer.addKey(key.xpub, client.mockHsm.signerConnection)
  return key.xpub
}).then(xpub =>
  Promise.all([
    client.assets.create({
      alias: 'gold',
      rootXpubs: [xpub],
      quorum: 1,
    }),
    client.accounts.create({
      alias: 'alice',
      rootXpubs: [xpub],
      quorum: 1
    }),
    client.accounts.create({
      alias: 'bob',
      rootXpubs: [xpub],
      quorum: 1
    })
  ])
).then(() =>
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 100
    })
  })
).then(
  template => signer.sign(template)
).then(
  signed => client.transactions.submit(signed)
).then(() => {
  // snippet create-receiver
  const aliceReceiverSerializedPromise = client.accounts.createReceiver({
    accountAlias: 'alice',
  }).then(aliceReceiver => {
    return JSON.stringify(aliceReceiver)
  })
  // endsnippet

  return aliceReceiverSerializedPromise
}).then(aliceReceiverSerialized => {
  // snippet build-transaction
  return client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 10
    })
    builder.controlWithReceiver({
      receiver: JSON.parse(aliceReceiverSerialized),
      assetAlias: 'gold',
      amount: 10
    })
  }).then(template => {
    return signer.sign(template)
  }).then(signed => {
    return client.transactions.submit(signed)
  })
  // endsnippet
}).then(() =>
  // snippet retire
  client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 10
    })
    builder.retire({
      assetAlias: 'gold',
      amount: 10
    })
  }).then(template => {
    return signer.sign(template)
  }).then(signed => {
    return client.transactions.submit(signed)
  })
  // endsnippet
).catch(err =>
  process.nextTick(() => { throw err })
)
