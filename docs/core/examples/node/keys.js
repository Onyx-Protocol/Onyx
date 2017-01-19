const chain = require('chain-sdk')

const client = new chain.Client()
let xpub
let _signer

Promise.resolve().then(() => {
  // snippet create-key
  const keyPromise = client.mockHsm.keys.create()
  // endsnippet

  return keyPromise
}).then(key => {
  xpub = key.xpub

  // snippet signer-add-key
  const signer = new chain.HsmSigner() // Holds multiple keys.
  signer.addKey(key.xpub, client.mockHsm.signerConnection)
  // endsnippet

  _signer = signer
}).then(() =>
  client.assets.create({
    alias: 'gold',
    rootXpubs: [xpub],
    quorum: 1,
  })
).then(() =>
  client.accounts.create({
    alias: 'alice',
    rootXpubs: [xpub],
    quorum: 1
  })
).then(() =>
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
  })
).then(unsigned => {
  const signer = _signer

  // snippet sign-transaction
  const signerPromise = signer.sign(unsigned)
  // endsnippet

  return signerPromise
}).then(signed =>
  client.transactions.submit(signed)
).catch(err =>
  process.nextTick(() => { throw err })
)
