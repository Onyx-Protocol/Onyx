const chain = require('chain-sdk')

// This demo is written to run on either one or two cores. Simply provide
// different URLs to the following clients for the two-core version.
const client = new chain.Client()
const otherClient = new chain.Client()

const signer = new chain.HsmSigner()
let aliceKey, bobKey

Promise.all([
  client.mockHsm.keys.create(),
  otherClient.mockHsm.keys.create()
]).then(keys => {
  aliceKey = keys[0].xpub,
  bobKey = keys[1].xpub

  signer.addKey(aliceKey, client.mockHsm.signerConnection)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    rootXpubs: [aliceKey],
    quorum: 1
  }),

  client.assets.create({
    alias: 'silver',
    rootXpubs: [aliceKey],
    quorum: 1
  }),

  client.accounts.create({
    alias: 'alice',
    rootXpubs: [aliceKey],
    quorum: 1
  }),

  otherClient.accounts.create({
    alias: 'bob',
    rootXpubs: [bobKey],
    quorum: 1
  })
])).then(() =>

  // snippet issue-within-core
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'silver',
      amount: 1000
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'silver',
      amount: 1000
    })
  })
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
  // endsnippet

).then(() =>
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'gold',
      amount: 1000
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 1000
    })
  })
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
).then(() =>

  // snippet create-bob-issue-receiver
  otherClient.accounts.createReceiver({
    accountAlias: 'bob'
  }).then(bobIssuanceReceiver => {
    return JSON.stringify(bobIssuanceReceiver)
  })
  // endsnippet

).then(bobIssuanceReceiverSerialized => {
  return (

    // snippet issue-to-bob-receiver
    client.transactions.build(builder => {
      builder.issue({
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithReceiver({
        receiver: JSON.parse(bobIssuanceReceiverSerialized),
        assetAlias: 'gold',
        amount: 10
      })
    })
    .then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  )
}).then(() => {
  if (client.baseUrl == otherClient.baseUrl){

    // snippet pay-within-core
    return client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithAccount({
        accountAlias: 'bob',
        assetAlias: 'gold',
        amount: 10
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  } else {
    return
  }
}).then(() =>

  // snippet create-bob-payment-receiver
  otherClient.accounts.createReceiver({
    accountAlias: 'bob'
  }).then(bobPaymentReceiver => {
    return JSON.stringify(bobPaymentReceiver)
  })
  // endsnippet

).then(bobPaymentReceiverSerialized => {
  return (

    // snippet pay-between-cores
    client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithReceiver({
        receiver: JSON.parse(bobPaymentReceiverSerialized),
        assetAlias: 'gold',
        amount: 10
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  )
}).then(() => {
  if (client.baseUrl == otherClient.baseUrl){

    //snippet multiasset-within-core
    return client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'silver',
        amount: 20
      })
      builder.controlWithAccount({
        accountAlias: 'bob',
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithAccount({
        accountAlias: 'bob',
        assetAlias: 'silver',
        amount: 20
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  } else {
    return
  }
}).then(() => {
  return (

    // snippet create-bob-multiasset-receiver
    Promise.all([
      otherClient.accounts.createReceiver({
        accountAlias: 'bob'
      }),
      otherClient.accounts.createReceiver({
        accountAlias: 'bob'
      }),
    ]).then(receivers => {
      return {
        bobGoldReceiverSerialized: JSON.stringify(receivers[0]),
        bobSilverReceiverSerialized: JSON.stringify(receivers[1]),
      }
    })
    // endsnippet

  )
}).then(({bobGoldReceiverSerialized, bobSilverReceiverSerialized}) => {
  return (

    // snippet multiasset-between-cores
    client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'silver',
        amount: 20
      })
      builder.controlWithReceiver({
        receiver: JSON.parse(bobGoldReceiverSerialized),
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithReceiver({
        receiver: JSON.parse(bobSilverReceiverSerialized),
        assetAlias: 'silver',
        amount: 20
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  )
}).then(() =>

  // snippet retire
  client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 50
    })
    builder.retire({
      assetAlias: 'gold',
      amount: 50
    })
  })
  .then(retirement => signer.sign(retirement))
  .then(signed => client.transactions.submit(signed))
  // endsnippet

).catch(err =>
  process.nextTick(() => { throw err })
)
