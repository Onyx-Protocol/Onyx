const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let asset_key, alice_key, bob_key

Promise.all([
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
]).then(keys => {
  asset_key = keys[0].xpub
  alice_key = keys[1].xpub
  bob_key   = keys[2].xpub

  signer.addKey(asset_key, client.mockHsm.signerUrl)
  signer.addKey(alice_key, client.mockHsm.signerUrl)
  signer.addKey(bob_key, client.mockHsm.signerUrl)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    root_xpubs: [asset_key],
    quorum: 1,
  }),
  client.assets.create({
    alias: 'silver',
    root_xpubs: [asset_key],
    quorum: 1,
  }),
  // snippet create-account-alice
  client.accounts.create({
    alias: 'alice',
    root_xpubs: [alice_key],
    quorum: 1,
    tags: {
      type: 'checking',
      first_name: 'Alice',
      last_name: 'Jones',
      user_id: '12345',
    }
  })
  // endsnippet
  ,
  // snippet create-account-bob
  client.accounts.create({
    alias: 'bob',
    root_xpubs: [bob_key],
    quorum: 1,
    tags: {
      type: 'savings',
      first_name: 'Bob',
      last_name: 'Smith',
      user_id: '67890',
    }
  })
  // endsnippet
])).then(() =>
  // snippet list-accounts-by-tag
  client.accounts.query({
    filter: 'tags.type=$1',
    filter_params: ['savings'],
  }).then(response => {
    for (let account of response) {
      console.log('Account ID ' + account.id + ' alias ' + account.alias)
    }
  })
  // endsnippet
).then(() =>
  client.transactions.build(function (builder) {
    builder.issue({ asset_alias: 'gold', amount: 100 })
    builder.issue({ asset_alias: 'silver', amount: 100 })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'silver',
      amount: 100
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() => {
  // snippet build-transfer
  const spendPromise = client.transactions.build(function (builder) {
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
  // endsnippet

  return spendPromise.then(spendingTx => {
    // snippet sign-transfer
    const signingPromise = signer.sign(spendingTx)
    // endsnippet

    return signingPromise
  }).then(signedSpendingTx =>
    // snippet submit-transfer
    client.transactions.submit(signedSpendingTx)
    // endsnippet
  )
}).then((submitted) => {
  // snippet create-control-program
  const bobProgramPromise = client.accounts.createControlProgram({
    alias: 'bob',
  })
  // endsnippet

  return bobProgramPromise.then(bobProgram =>
    // snippet transfer-to-control-program
    client.transactions.build(function (builder) {
        builder.spendFromAccount({
          account_alias: 'alice',
          asset_alias: 'gold',
          amount: 10
        })
        builder.controlWithProgram({
          control_program: bobProgram.control_program,
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
}).then(() =>
  // snippet list-account-txs
  client.transactions.query({
    filter: 'inputs(account_alias=$1) AND outputs(account_alias=$1)',
    filter_params: ['alice'],
  }).then(response => {
    for (let transaction of response) {
      console.log(transaction.id + ' at ' + transaction.timestamp)
    }
  })
  // endsnippet
).then(() =>
  // snippet list-account-balances
  client.balances.query({
    filter: 'account_alias=$1',
    filter_params: ['alice'],
  }).then(response => {
    for (let balance of response) {
      console.log("Alice's balance of " + balance.sum_by.asset_alias + ': ' + balance.amount)
    }
  })
  // endsnippet
).then(() =>
  // snippet list-account-unspent-outputs
  client.unspentOutputs.query({
    filter: 'account_alias=$1 AND asset_alias=$2',
    filter_params: ['alice', 'gold'],
  }).then(response => {
    for (let unspent of response) {
      console.log(unspent.transaction_id + ' position ' + unspent.position)
    }
  })
  // endsnippet
)
