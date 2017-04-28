const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

Promise.all([
  client.mockHsm.keys.create(),
]).then(keys => {
  key  = keys[0].xpub

  signer.addKey(key, client.mockHsm.signerConnection)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    rootXpubs: [key],
    quorum: 1
  }),
  client.assets.create({
    alias: 'silver',
    rootXpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'alice',
    tags: {
      type: 'checking'
    },
    rootXpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    rootXpubs: [key],
    quorum: 1
  })
])
).then(() =>
  client.transactions.build(builder => {
    builder.issue({ assetAlias: 'gold', amount: 1000 })
    builder.issue({ assetAlias: 'silver', amount: 1000 })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 1000
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'silver',
      amount: 1000
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() =>
  client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'alice',
      assetAlias: 'gold',
      amount: 10
    })
    builder.spendFromAccount({
      accountAlias: 'bob',
      assetAlias: 'silver',
      amount: 10
    })
    builder.controlWithAccount({
      accountAlias: 'alice',
      assetAlias: 'silver',
      amount: 10
    })
    builder.controlWithAccount({
      accountAlias: 'bob',
      assetAlias: 'gold',
      amount: 10
    })
  }).then(trade => signer.sign(trade))
    .then(signed => client.transactions.submit(signed))
).then(() => Promise.all([
  client.assets.create({
    alias: 'bank1UsdIou',
    rootXpubs: [key],
    quorum: 1,
    definition: {
      currency: 'USD'
    }
  }),
  client.assets.create({
    alias: 'bank1EuroIou',
    rootXpubs: [key],
    quorum: 1,
    definition: {
      currency: 'Euro'
    }
  }),
  client.assets.create({
    alias: 'bank2UsdIou',
    rootXpubs: [key],
    quorum: 1,
    definition: {
      currency: 'USD'
    }
  }),
  client.accounts.create({
    alias: 'bank1',
    rootXpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bank2',
    rootXpubs: [key],
    quorum: 1
  })
])
).then(() =>
  client.transactions.build(builder => {
    builder.issue({ assetAlias: 'bank1UsdIou', amount: 2000000 })
    builder.issue({ assetAlias: 'bank2UsdIou', amount: 2000000 })
    builder.issue({ assetAlias: 'bank1EuroIou', amount: 2000000 })
    builder.controlWithAccount({
      accountAlias: 'bank1',
      assetAlias: 'bank1UsdIou',
      amount: 1000000
    })
    builder.controlWithAccount({
      accountAlias: 'bank1',
      assetAlias: 'bank1EuroIou',
      amount: 1000000
    })
    builder.controlWithAccount({
      accountAlias: 'bank1',
      assetAlias: 'bank2UsdIou',
      amount: 1000000
    })
    builder.controlWithAccount({
      accountAlias: 'bank2',
      assetAlias: 'bank1UsdIou',
      amount: 1000000
    })
    builder.controlWithAccount({
      accountAlias: 'bank2',
      assetAlias: 'bank1EuroIou',
      amount: 1000000
    })
    builder.controlWithAccount({
      accountAlias: 'bank2',
      assetAlias: 'bank2UsdIou',
      amount: 1000000
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() =>

  // snippet list-alice-transactions
  client.transactions.queryAll({
    filter: 'inputs(account_alias=$1) OR outputs(account_alias=$1)',
    filterParams: ['alice'],
  }, (tx, next, done) => {
    console.log("Alice's transaction: " + tx.id)

    tx.inputs.forEach(input => {
      console.log('-' + input.amount + ' ' + input.assetAlias)
    })

    tx.outputs.forEach(output => {
      console.log('+' + output.amount + ' ' + output.assetAlias)
    })

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-checking-transactions
  client.transactions.queryAll({
    filter: 'inputs(account_tags.type=$1) OR outputs(account_tags.type=$1)',
    filterParams: ['checking'],
  }, (tx, next, done) => {
    console.log("Checking account transaction: " + tx.id)

    tx.inputs.forEach(input => {
      console.log('-' + input.amount + ' ' + input.assetAlias)
    })

    tx.outputs.forEach(output => {
      console.log('+' + output.amount + ' ' + output.assetAlias)
    })

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-local-transactions
  client.transactions.queryAll({
    filter: 'is_local=$1',
    filterParams: ['yes']
  }, (tx, next, done) => {
    console.log('Local transaction ' + tx.id)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-local-assets
  client.assets.queryAll({
    filter: 'is_local=$1',
    filterParams: ['yes']
  }, (asset, next, done) => {
    console.log('Local asset ' + asset.id + ' (' + asset.alias + ')')

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-usd-assets
  client.assets.queryAll({
    filter: 'definition.currency=$1',
    filterParams: ['USD']
  }, (asset, next, done) => {
    console.log('USD asset ' + asset.id + ' (' + asset.alias + ')')

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-checking-accounts
  client.accounts.queryAll({
    filter: 'tags.type=$1',
    filterParams: ['checking']
  }, (account, next, done) => {
    console.log('Checking account ' + account.id + ' (' + account.alias + ')')

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-alice-unspents
  client.unspentOutputs.queryAll({
    filter: 'account_alias=$1',
    filterParams: ['alice']
  }, (utxo, next, done) => {
    console.log("Alice's unspent output: " + utxo.amount + ' ' + utxo.assetAlias)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet list-checking-unspents
  client.unspentOutputs.queryAll({
    filter: 'account_tags.type=$1',
    filterParams: ['checking']
  }, (utxo, next, done) => {
    console.log("Checking account unspent output: " + utxo.amount + ' ' + utxo.assetAlias)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet account-balance
  client.balances.queryAll({
    filter: 'account_alias=$1',
    filterParams: ['bank1']
  }, (balance, next, done) => {
    console.log('Bank 1 balance of ' + balance.sumBy.assetAlias + ': ' + balance.amount)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet checking-accounts-balance
  client.balances.queryAll({
    filter: 'account_tags.type=$1',
    filterParams: ['checking']
  }, (balance, next, done) => {
    console.log('Checking accounts balance of ' + balance.sumBy.assetAlias + ': ' + balance.amount)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet usd-iou-circulation
  client.balances.queryAll({
    filter: 'asset_alias=$1',
    filterParams: ['bank1UsdIou']
  }, (balance, next, done) => {
    console.log('Total circulation of Bank 1 USD IOU: ' + balance.amount)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).then(() =>

  // snippet account-balance-sum-by-currency
  client.balances.queryAll({
    filter: 'account_alias=$1',
    filterParams: ['bank1'],
    sumBy: ['asset_definition.currency']
  }, (balance, next, done) => {
    var denom = balance.sumBy['assetDefinition.currency']
    console.log('Bank 1 balance of ' + denom + '-denominated currencies: ' + balance.amount)

    // next() moves to the next item.
    // done() terminates the loop early, and causes the
    //   query promise to resolve. Passing an error will reject
    //   the promise.
    next()
  })
  // endsnippet

).catch(err =>
  process.nextTick(() => { throw err })
)
