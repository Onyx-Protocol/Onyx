const chain = require('chain-sdk')

// This demo is written to run on either one or two cores. Simply provide
// different URLs to the following clients for the two-core version.
const client = new chain.Client()
const otherClient = new chain.Client()

const signer = new chain.HsmSigner()
let aliceKey, bobKey, bobProgram

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

  // snippet create bobIssueProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  })
  // endsnippet

).then(program => bobProgram = program.controlProgram)
  .then(() =>

  // snippet issueToBobProgram
  client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'gold',
      amount: 10
    })
    builder.controlWithProgram({
      controlProgram: bobProgram,
      assetAlias: 'gold',
      amount: 10
    })
  })
  .then(issuance => signer.sign(issuance))
  .then(signed => client.transactions.submit(signed))
  // endsnippet

).then(() => {
  if (client.baseUrl == otherClient.baseUrl){

    // snippet payWithinCore
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

  // snippet createBobPaymentProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  }))
  // endsnippet

  .then(program => bobProgram = program.controlProgram)
    .then(() =>

    // snippet payBetweenCores
    client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithProgram({
        controlProgram: bobProgram,
        assetAlias: 'gold',
        amount: 10
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

).then(() => {
  if (client.baseUrl == otherClient.baseUrl){

    //snippet multiAssetWithinCore
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
}).then(() =>

  // snippet createBobMultiAssetProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  }))
  // endsnippet

  .then(program => bobProgram = program.controlProgram)
  .then(() =>

    // snippet multiAssetBetweenCores
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
      builder.controlWithProgram({
        controlProgram: bobProgram,
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithProgram({
        controlProgram: bobProgram,
        assetAlias: 'silver',
        amount: 20
      })
    })
    .then(payment => signer.sign(payment))
    .then(signed => client.transactions.submit(signed))
    // endsnippet

  ).then(() =>

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
