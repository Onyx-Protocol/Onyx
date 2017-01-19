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
  client.transactions.query({
    filter: 'inputs(account_alias=$1) OR outputs(account_alias=$1)',
    filterParams: ['alice'],
  }).then(results =>
    results.items.forEach(tx => {
      console.log("Alice's transaction: " + tx.id)

      tx.inputs.forEach(input => {
        console.log('-' + input.amount + ' ' + input.assetAlias)
      })

      tx.outputs.forEach(output => {
        console.log('+' + output.amount + ' ' + output.assetAlias)
      })
    })
  )
  // endsnippet

).then(() =>

  // snippet list-local-transactions
  client.transactions.query({
    filter: 'is_local=$1',
    filterParams: ['yes']
  }).then(results =>
    results.items.forEach(tx => {
      console.log('Local transaction ' + tx.id)
    })
  )
  // endsnippet

).then(() =>

  // snippet list-local-assets
  client.assets.query({
    filter: 'is_local=$1',
    filterParams: ['yes']
  }).then(results =>
    results.items.forEach(asset => {
      console.log('Local asset ' + asset.id + ' (' + asset.alias + ')')
    })
  )
  // endsnippet

).then(() =>

  // snippet list-usd-assets
  client.assets.query({
    filter: 'definition.currency=$1',
    filterParams: ['USD']
  }).then(results =>
    results.items.forEach(asset => {
      console.log('USD asset ' + asset.id + ' (' + asset.alias + ')')
    })
  )
  // endsnippet

).then(() =>

  // snippet list-checking-accounts
  client.accounts.query({
    filter: 'tags.type=$1',
    filterParams: ['checking']
  }).then(results =>
    results.items.forEach(account => {
      console.log('Checking account account ' + account.id + ' (' + account.alias + ')')
    })
  )
  // endsnippet

).then(() =>

  // snippet list-alice-unspents
  client.unspentOutputs.query({
    filter: 'account_alias=$1',
    filterParams: ['alice']
  }).then(results =>
    results.items.forEach(utxo => {
      console.log("Alice's unspent output: " + utxo.amount + ' ' + utxo.assetAlias)
    })
  )
  // endsnippet

).then(() =>

  // snippet account-balance
  client.balances.query({
    filter: 'account_alias=$1',
    filterParams: ['bank1']
  }).then(results =>
    results.items.forEach(b => {
      console.log('Bank 1 balance of ' + b.sumBy.assetAlias + ': ' + b.amount)
    })
  )
  // endsnippet

).then(() =>

  // snippet usd-iou-circulation
  client.balances.query({
    filter: 'asset_alias=$1',
    filterParams: ['bank1UsdIou']
  }).then(results =>
    results.items.forEach(b => {
      console.log('Total circulation of Bank 1 USD IOU: ' + b.amount)
    })
  )
  // endsnippet

).then(() =>

  // snippet account-balance-sum-by-currency
  client.balances.query({
    filter: 'account_alias=$1',
    filterParams: ['bank1'],
    sumBy: ['assetDefinition.currency']
  }).then(results =>
    results.items.forEach(b => {
      var denom = b.sumBy['assetDefinition.currency']
      console.log('Bank 1 balance of ' + denom + '-denominated currencies: ' + b.amount)
    })
  )
  // endsnippet

).catch(err =>
  process.nextTick(() => { throw err })
)
