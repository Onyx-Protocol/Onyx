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
  signer.addKey(key.xpub, client.mockHsm.signerUrl)
  // endsnippet

  _signer = signer
}).then(() =>
  client.assets.create({
    alias: 'gold',
    root_xpubs: [xpub],
    quorum: 1,
  })
).then(() =>
  client.accounts.create({
     alias: 'alice',
     root_xpubs: [xpub],
     quorum: 1
   })
).then(() =>
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
  })
).then(unsigned => {
  const signer = _signer

  // snippet sign-transaction
  const signerPromise = signer.sign(unsigned)
  // endsnippet

  return signerPromise
}).then(signed =>
  client.transactions.submit(signed)
)
