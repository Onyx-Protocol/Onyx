const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let assetKey, aliceKey, bobKey

Promise.all([
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
]).then(keys => {
  assetKey = keys[0].xpub
  aliceKey = keys[1].xpub
  bobKey   = keys[2].xpub

  signer.addKey(assetKey, client.mockHsm.signerConnection)
  signer.addKey(aliceKey, client.mockHsm.signerConnection)
  signer.addKey(bobKey, client.mockHsm.signerConnection)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    rootXpubs: [assetKey],
    quorum: 1,
  }),
  client.assets.create({
    alias: 'silver',
    rootXpubs: [assetKey],
    quorum: 1,
  }),
  // snippet create-account-alice
  client.accounts.create({
    alias: 'alice',
    rootXpubs: [aliceKey],
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
    rootXpubs: [bobKey],
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
  client.accounts.queryAll({
    filter: 'tags.type=$1',
    filterParams: ['savings'],
  }, (account, next) => {
    console.log('Account ID ' + account.id + ' alias ' + account.alias)
    next()
  })
  // endsnippet
).then(() =>
  client.transactions.build(builder => {
    builder.issue({ assetAlias: 'gold', amount: 100 })
    builder.issue({ assetAlias: 'silver', amount: 100 })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 100
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'silver',
      amount: 100
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() => {
  // snippet build-transfer
  const spendPromise = client.transactions.build(builder => {
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
}).then(() => {
  // snippet create-receiver
  const bobReceiverSerializedPromise = client.accounts.createReceiver({
    accountAlias: 'bob',
  }).then(bobReceiver => {
    return JSON.stringify(bobReceiver)
  })
  // endsnippet

  return bobReceiverSerializedPromise.then(bobReceiverSerialized =>
    // snippet transfer-to-receiver
    client.transactions.build(builder => {
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10
      })
      builder.controlWithReceiver({
        receiver: JSON.parse(bobReceiverSerialized),
        assetAlias: 'gold',
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
  client.transactions.queryAll({
    filter: 'inputs(account_alias=$1) AND outputs(account_alias=$1)',
    filterParams: ['alice'],
  }, (transaction, next) => {
    console.log(transaction.id + ' at ' + transaction.timestamp)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-account-balances
  client.balances.queryAll({
    filter: 'account_alias=$1',
    filterParams: ['alice'],
  }, (balance, next) => {
    console.log("Alice's balance of " + balance.sumBy.assetAlias + ': ' + balance.amount)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-account-unspent-outputs
  client.unspentOutputs.queryAll({
    filter: 'account_alias=$1 AND asset_alias=$2',
    filterParams: ['alice', 'gold'],
  }, (unspent, next) => {
    console.log('Output ID: ' + unspent.id)
    next()
  })
  // endsnippet
).catch(err =>
  process.nextTick(() => { throw err })
)
