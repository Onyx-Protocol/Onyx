const chain = require('chain-sdk')

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
  signer.addKey(key.xpub, client.mockHsm.signerConnection)
  // endsnippet

  _signer = signer
  return key
}).then(key => {
  // snippet create-asset
  const goldPromise = client.assets.create({
    alias: 'gold',
    rootXpubs: [key.xpub],
    quorum: 1,
  })
  // endsnippet

  // snippet create-account-alice
  const alicePromise = client.accounts.create({
    alias: 'alice',
    rootXpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet

  // snippet create-account-bob
  const bobPromise = client.accounts.create({
    alias: 'bob',
    rootXpubs: [key.xpub],
    quorum: 1
  })
  // endsnippet

  return Promise.all([goldPromise, alicePromise, bobPromise])
}).then(() => {
  const signer = _signer

  Promise.resolve().then(() =>

    // snippet issue
    client.transactions.build(builder => {
      builder.issue({
        assetAlias: 'gold',
        amount: 100
      })
      builder.controlWithAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
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
    client.transactions.build(builder => {
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
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet

  ).then(() =>

    // snippet retire
    client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 5
      })
      builder.retire({
        assetAlias: 'gold',
        amount: 5
      })
    }).then(issuance => {
      return signer.sign(issuance)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet
  )
}).catch(err =>
  process.nextTick(() => { throw err })
)
