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

  signer.addKey(aliceKey, client.mockHsm.signerUrl)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    root_xpubs: [aliceKey],
    quorum: 1
  }),

  client.assets.create({
    alias: 'silver',
    root_xpubs: [aliceKey],
    quorum: 1
  }),

  client.accounts.create({
    alias: 'alice',
    root_xpubs: [aliceKey],
    quorum: 1
  }),

  otherClient.accounts.create({
    alias: 'bob',
    root_xpubs: [bobKey],
    quorum: 1
  })
])).then(() =>

  // snippet issue-within-core
  client.transactions.build( function(builder){
    builder.issue({
      asset_alias: 'silver',
      amount: 1000
    })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'silver',
      amount: 1000
    })
  })
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
  // endsnippet

).then(() =>
  client.transactions.build( function(builder){
    builder.issue({
      asset_alias: 'gold',
      amount: 1000
    })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 1000
    })
  })
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
).then(() =>

  // snippet create bobIssueProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  })
  // endsnippet

).then((program) => bobProgram = program.control_program)
  .then(() =>

  // snippet issueToBobProgram
  client.transactions.build( function(builder){
    builder.issue({
      asset_alias: 'gold',
      amount: 10
    })
    builder.controlWithProgram({
      control_program: bobProgram,
      asset_alias: 'gold',
      amount: 10
    })
  })
  .then((issuance) => signer.sign(issuance))
  .then((signed) => client.transactions.submit(signed))
  // endsnippet

).then(() => {
    if (client.baseUrl == otherClient.baseUrl){

      // snippet payWithinCore
      return client.transactions.build( function(builder){
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'gold',
          amount: 10
        })
        builder.controlWithAccount({
          account_alias: 'bob',
          asset_alias: 'gold',
          amount: 10
        })
      })
      .then((payment) => signer.sign(payment))
      .then((signed) => client.transactions.submit(signed))
      // endsnippet

    } else {
      return
    }
  }
).then(() =>

  // snippet createBobPaymentProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  }))
  // endsnippet

  .then((program) => bobProgram = program.control_program)
    .then(() =>

    // snippet payBetweenCores
    client.transactions.build( function(builder){
      builder.spendFromAccount({
        account_alias: 'alice',
        asset_alias: 'gold',
        amount: 10
      })
      builder.controlWithProgram({
        control_program: bobProgram,
        asset_alias: 'gold',
        amount: 10
      })
    })
    .then((payment) => signer.sign(payment))
    .then((signed) => client.transactions.submit(signed))
    // endsnippet

).then(() => {
    if (client.baseUrl == otherClient.baseUrl){

      //snippet multiAssetWithinCore
      return client.transactions.build( function(builder){
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'gold',
          amount: 10
        })
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'silver',
          amount: 20
        })
        builder.controlWithAccount({
          account_alias: 'bob',
          asset_alias: 'gold',
          amount: 10
        })
        builder.controlWithAccount({
          account_alias: 'bob',
          asset_alias: 'silver',
          amount: 20
        })
      })
      .then((payment) => signer.sign(payment))
      .then((signed) => client.transactions.submit(signed))
      // endsnippet

    } else {
      return
    }
  }
).then(() =>

  // snippet createBobMultiAssetProgram
  otherClient.accounts.createControlProgram({
    alias: 'bob'
  }))
  // endsnippet

  .then((program) => bobProgram = program.control_program)
  .then(() =>

    // snippet multiAssetBetweenCores
    client.transactions.build( function(builder){
      builder.spendFromAccount({
        account_alias: 'alice',
        asset_alias: 'gold',
        amount: 10
      })
      builder.spendFromAccount({
        account_alias: 'alice',
        asset_alias: 'silver',
        amount: 20
      })
      builder.controlWithProgram({
        control_program: bobProgram,
        asset_alias: 'gold',
        amount: 10
      })
      builder.controlWithProgram({
        control_program: bobProgram,
        asset_alias: 'silver',
        amount: 20
      })
    })
    .then((payment) => signer.sign(payment))
    .then((signed) => client.transactions.submit(signed))
    // endsnippet

  ).then(() =>

  // snippet retire
  client.transactions.build( function(builder){
    builder.spendFromAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 50
    })
    builder.retire({
      asset_alias: 'gold',
      amount: 50
    })
  })
  .then((retirement) => signer.sign(retirement))
  .then((signed) => client.transactions.submit(signed))
  // endsnippet
)
