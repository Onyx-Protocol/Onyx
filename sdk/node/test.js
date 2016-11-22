const chain = require('./index')

// snippet create-client
const client = new chain.Client()
// endsnippet

let _signer

Promise.resolve().then(() => {
  // snippet create-key
  const keyPromise = client.mockHsm.keys.create()
  // endsnippet

  return keyPromise
}).then(key => {
  // snippet signer-add-key
  const signer = new chain.HsmSigner()
  signer.addKey(key.xpub, client.mockHsm.signerUrl)
  // endsnippet

  _signer = signer
  return key
}).then(key => {
  // snippet create-asset
  const goldPromise = client.assets.create({
    alias: 'gold',
    root_xpubs: [key.xpub],
    quorum: 1,
  })
  // endsnippet

  // snippet create-account-alice
  const alicePromise = client.accounts.create({
    alias: 'alice',
    root_xpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet

  // snippet create-account-bob
  const bobPromise = client.accounts.create({
    alias: 'bob',
    root_xpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet

  return Promise.all([goldPromise, alicePromise, bobPromise])
}).then(() => {
  const signer = _signer

  Promise.resolve().then(() =>

    // snippet issue
    client.transactions.build(function (builder) {
      builder.issue({
        asset_alias: 'gold',
        amount: 100
      })
      builder.controlWithAccount({
        account_alias: 'alice',
        asset_alias: 'gold',
        amount: 100
      })
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet

  ).then(() =>

    // snippet spend
    client.transactions.build(function (builder) {
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
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet

  ).then(() =>

    // snippet retire
    client.transactions.build(function (builder) {
      builder.spendFromAccount({
        account_alias: 'alice',
        asset_alias: 'gold',
        amount: 5
      })
      builder.retire({
        asset_alias: 'gold',
        amount: 5
      })
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet

  )
})
