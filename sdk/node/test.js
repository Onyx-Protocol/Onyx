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
  signer.addKey(key, client.mockHsm.signerUrl)
  // endsnippet

  // _signer = signer
  return key
}).then(key => {
  // snippet create-asset
  client.assets.create({
    alias: 'gold',
    root_xpubs: [key.xpub],
    quorum: 1,
  })
  // endsnippet

  // snippet create-account-alice
  client.accounts.create({
    alias: 'alice',
    root_xpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet

  // snippet create-account-bob
  client.accounts.create({
    alias: 'bob',
    root_xpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet
}).then(() => {
  // snippet issue
  client.transactions.build(function (builder) {
    builder.issue({
      asset_alias: 'gold',
      amount: 100
    })
    build.controlWithAccount({
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


  // snippet spend
  // const spending = client.transactions.build(function (builder) {
  //   builder.spendFromAccount({
  //     account_alias: 'alice'
  //     asset_alias: 'gold',
  //     amount: 10
  //   })
  //   build.controlWithAccount({
  //     account_alias: 'bob',
  //     asset_alias: 'gold',
  //     amount: 10
  //   })
  // })
  //
  // client.transactions.submit(signer.sign(spending))
  // endsnippet
  //
  // snippet retire
  // const retirement = client.transactions.build(function (builder) {
  //   builder.spendFromAccount({
  //     account_alias: 'alice'
  //     asset_alias: 'gold',
  //     amount: 5
  //   })
  //   build.retire({
  //     asset_alias: 'gold',
  //     amount: 5
  //   })
  // })
  //
  // client.transactions.submit(signer.sign(retirement))
  // endsnippet
})
