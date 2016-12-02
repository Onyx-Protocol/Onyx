const chain = require('chain-sdk')

// This demo is written to run on either one or two cores. Simply provide
// different URLs to the following clients for the two-core version.
const aliceCore = new chain.Client()
const bobCore = new chain.Client()

const aliceSigner = new chain.HsmSigner()
const bobSigner = new chain.HsmSigner()

let aliceDollarKey, bobBuckKey, aliceKey, bobKey, aliceDollar, bobBuck

Promise.all([
  aliceCore.mockHsm.keys.create(),
  bobCore.mockHsm.keys.create(),
  aliceCore.mockHsm.keys.create(),
  bobCore.mockHsm.keys.create(),
]).then(keys => {
  aliceDollarKey  = keys[0].xpub
  bobBuckKey      = keys[1].xpub
  aliceKey        = keys[2].xpub
  bobKey          = keys[3].xpub

  aliceSigner.addKey(aliceDollarKey, aliceCore.mockHsm.signerUrl)
  bobSigner.addKey(bobBuckKey, bobCore.mockHsm.signerUrl)
  aliceSigner.addKey(aliceKey, aliceCore.mockHsm.signerUrl)
  bobSigner.addKey(bobKey, bobCore.mockHsm.signerUrl)
}).then(() => Promise.all([

  // snippet create-asset-aliceDollar
  aliceCore.assets.create({
    alias: 'alice_dollar',
    root_xpubs: [aliceDollarKey],
    quorum: 1,
  }),
  // endsnippet

  // create-asset-bobBuck
  bobCore.assets.create({
    alias: 'bob_buck',
    root_xpubs: [bobBuckKey],
    quorum: 1,
  }),
  // endsnippet

  // snippet create-account-alice
  aliceCore.accounts.create({
    alias: 'alice',
    root_xpubs: [aliceKey],
    quorum: 1,
  }),
  // endsnippet

  // snippet create-account-bob
  bobCore.accounts.create({
    alias: 'bob',
    root_xpubs: [bobKey],
    quorum: 1,
  })
  // endsnippet

])).then(assets => {
  aliceDollar = assets[0]
  bobBuck = assets[1]
}).then(() =>
  aliceCore.transactions.build(function (builder) {
    builder.issue({ asset_alias: 'alice_dollar', amount: 1000 })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'alice_dollar',
      amount: 1000
    })
  }).then(issuance => aliceSigner.sign(issuance))
    .then(signed => aliceCore.transactions.submit(signed))
).then(() =>
  bobCore.transactions.build(function (builder) {
    builder.issue({ asset_alias: 'bob_buck', amount: 1000 })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'bob_buck',
      amount: 1000
    })
  }).then(issuance => bobSigner.sign(issuance))
    .then(signed => bobCore.transactions.submit(signed))
).then(() => {
    if (aliceCore.baseUrl == bobCore.baseUrl){
      const chain = aliceCore
      const signer = aliceSigner
      signer.addKey(bobKey, chain.mockHsm.signerUrl)

      // SAME-CORE TRADE

      // snippet same-core-trade
      chain.transactions.build(function (builder) {
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'alice_dollar',
          amount: 50
        })
        builder.controlWithAccount({
          account_alias: 'alice',
          asset_alias: 'bob_buck',
          amount: 100
        })
        builder.spendFromAccount({
          account_alias: 'bob',
          asset_alias: 'bob_buck',
          amount: 100
        })
        builder.controlWithAccount({
          account_alias: 'bob',
          asset_alias: 'alice_dollar',
          amount: 50
        })
      }).then(trade => signer.sign(trade))
        .then(signed => chain.transactions.submit(signed))
      // endsnippet

    } else {
      // CROSS-CORE TRADE

      const aliceDollarAssetId = aliceDollar.id
      const bobBuckAssetId = bobBuck.id

      // snippet build-trade-alice
      aliceCore.transactions.build(function (builder) {
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'alice_dollar',
          amount: 50
        })
        builder.controlWithAccount({
          account_alias: 'alice',
          asset_id: bobBuckAssetId,
          amount: 100
        })
      })
      // endsnippet

        // snippet sign-trade-alice
        .then(aliceTrade => {
          aliceTrade.allow_additional_actions = true
          return aliceSigner.sign(aliceTrade)
        })
        // endsnippet

        .then(aliceSigned =>

          // snippet build-trade-bob
          bobCore.transactions.build(function (builder) {
            builder.baseTransaction(aliceSigned.raw_transaction)
            builder.spendFromAccount({
              account_alias: 'bob',
              asset_alias: 'bob_buck',
              amount: 100
            })
            builder.controlWithAccount({
              account_alias: 'bob',
              asset_id: aliceDollarAssetId,
              amount: 50
            })
          })
          // endsnippet

            // snippet sign-trade-bob
            .then(bobTrade => bobSigner.sign(bobTrade))
            // endsnippet

            // snippet submit-trade-bob
            .then(bobSigned => bobCore.transactions.submit(bobSigned))
            // endsnippet
      )
    }
  }
)
