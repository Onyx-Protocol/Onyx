const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()

client.mockHsm.keys.create().then(key => {
  signer.addKey(key.xpub, client.mockHsm.signerUrl)
  return key.xpub
}).then(xpub =>
  Promise.all([
    client.assets.create({
      alias: 'gold',
      root_xpubs: [xpub],
      quorum: 1,
    }),
    client.accounts.create({
      alias: 'alice',
      root_xpubs: [xpub],
      quorum: 1
    }),
    client.accounts.create({
      alias: 'bob',
      root_xpubs: [xpub],
      quorum: 1
    })
  ])
).then(() =>
  client.transactions.build(function (builder) {
    builder.issue({
      asset_alias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 100
    })
  })
).then(
  template => signer.sign(template)
).then(
  signed => client.transactions.submit(signed)
).then(() => {
  // snippet create-control-program
  const aliceProgramPromise = client.accounts.createControlProgram({
    alias: 'alice',
  })
  // endsnippet

  return aliceProgramPromise
}).then(aliceProgram => {
  // snippet build-transaction
  return client.transactions.build(function (builder) {
    builder.spendFromAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 10
    })
    builder.controlWithProgram({
      control_program: aliceProgram.control_program,
      asset_alias: 'gold',
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
  client.transactions.build(function (builder) {
    builder.spendFromAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 10
    })
    builder.retire({
      asset_alias: 'gold',
      amount: 10
    })
  }).then(template => {
    return signer.sign(template)
  }).then(signed => {
    return client.transactions.submit(signed)
  })
  // endsnippet
)
