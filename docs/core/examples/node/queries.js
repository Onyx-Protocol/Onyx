const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let key

Promise.all([
  client.mockHsm.keys.create(),
]).then(keys => {
  key  = keys[0].xpub

  signer.addKey(key, client.mockHsm.signerUrl)
}).then(() => Promise.all([
  client.assets.create({
    alias: 'gold',
    root_xpubs: [key],
    quorum: 1
  }),
  client.assets.create({
    alias: 'silver',
    root_xpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'alice',
    tags: {
      type: 'checking'
    },
    root_xpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bob',
    root_xpubs: [key],
    quorum: 1
  })
])
).then(() =>
  client.transactions.build(function (builder) {
    builder.issue({ asset_alias: 'gold', amount: 1000 })
    builder.issue({ asset_alias: 'silver', amount: 1000 })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 1000
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'silver',
      amount: 1000
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() =>
  client.transactions.build(function (builder) {
    builder.spendFromAccount({
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 10
    })
    builder.spendFromAccount({
      account_alias: 'bob',
      asset_alias: 'silver',
      amount: 10
    })
    builder.controlWithAccount({
      account_alias: 'alice',
      asset_alias: 'silver',
      amount: 10
    })
    builder.controlWithAccount({
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 10
    })
  }).then(trade => signer.sign(trade))
    .then(signed => client.transactions.submit(signed))
).then(() => Promise.all([
  client.assets.create({
    alias: 'bank1_usd_iou',
    root_xpubs: [key],
    quorum: 1,
    definition: {
      currency: 'USD'
    }
  }),
  client.assets.create({
    alias: 'bank1_euro_iou',
    root_xpubs: [key],
    quorum: 1,
    definition: {
      currency: 'Euro'
    }
  }),
  client.assets.create({
    alias: 'bank2_usd_iou',
    root_xpubs: [key],
    quorum: 1,
    definition: {
      currency: 'USD'
    }
  }),
  client.accounts.create({
    alias: 'bank1',
    root_xpubs: [key],
    quorum: 1
  }),
  client.accounts.create({
    alias: 'bank2',
    root_xpubs: [key],
    quorum: 1
  })
])
).then(() =>
  client.transactions.build(function (builder) {
    builder.issue({ asset_alias: 'bank1_usd_iou', amount: 2000000 })
    builder.issue({ asset_alias: 'bank2_usd_iou', amount: 2000000 })
    builder.issue({ asset_alias: 'bank1_euro_iou', amount: 2000000 })
    builder.controlWithAccount({
      account_alias: 'bank1',
      asset_alias: 'bank1_usd_iou',
      amount: 1000000
    })
    builder.controlWithAccount({
      account_alias: 'bank1',
      asset_alias: 'bank1_euro_iou',
      amount: 1000000
    })
    builder.controlWithAccount({
      account_alias: 'bank1',
      asset_alias: 'bank2_usd_iou',
      amount: 1000000
    })
    builder.controlWithAccount({
      account_alias: 'bank2',
      asset_alias: 'bank1_usd_iou',
      amount: 1000000
    })
    builder.controlWithAccount({
      account_alias: 'bank2',
      asset_alias: 'bank1_euro_iou',
      amount: 1000000
    })
    builder.controlWithAccount({
      account_alias: 'bank2',
      asset_alias: 'bank2_usd_iou',
      amount: 1000000
    })
  }).then(issuance => signer.sign(issuance))
    .then(signed => client.transactions.submit(signed))
).then(() =>

  // snippet list-alice-transactions
  client.transactions.query({
    filter: 'inputs(account_alias=$1) OR outputs(account_alias=$1)',
    filter_params: ['alice'],
  }).then((results) =>
    results.items.forEach((tx) => {
      console.log("Alice's transaction: " + tx.id)

      tx.inputs.forEach((input) => {
        console.log("-" + input.amount + " " + input.asset_alias)
      })

      tx.outputs.forEach((output) => {
        console.log("+" + output.amount + " " + output.asset_alias)
      })
    })
  )
  // endsnippet

).then(() =>

  // snippet list-local-transactions
  client.transactions.query({
    filter: 'is_local=$1',
    filter_params: ['yes']
  }).then((results) =>
    results.items.forEach((tx) => {
      console.log("Local transaction " + tx.id)
    })
  )
  // endsnippet

).then(() =>

  // snippet list-local-assets
  client.assets.query({
    filter: 'is_local=$1',
    filter_params: ['yes']
  }).then((results) =>
    results.items.forEach((asset) => {
      console.log("Local asset " + asset.id + " (" + asset.alias + ")")
    })
  )
  // endsnippet

).then(() =>

  // snippet list-usd-assets
  client.assets.query({
    filter: 'definition.currency=$1',
    filter_params: ['USD']
  }).then((results) =>
    results.items.forEach((asset) => {
      console.log("USD asset " + asset.id + " (" + asset.alias + ")")
    })
  )
  // endsnippet

).then(() =>

  // snippet list-checking-accounts
  client.accounts.query({
    filter: 'tags.type=$1',
    filter_params: ['checking']
  }).then((results) =>
    results.items.forEach((account) => {
      console.log("Checking account account " + account.id + " (" + account.alias + ")")
    })
  )
  // endsnippet

).then(() =>

  // snippet list-alice-unspents
  client.unspentOutputs.query({
    filter: 'account_alias=$1',
    filter_params: ['alice']
  }).then((results) =>
    results.items.forEach((utxo) => {
      console.log("Alice's unspent output: " + utxo.amount + " " + utxo.asset_alias)
    })
  )
  // endsnippet

).then(() =>

  // snippet account-balance
  client.balances.query({
    filter: 'account_alias=$1',
    filter_params: ['bank1']
  }).then((results) =>
    results.items.forEach((b) => {
      console.log("Bank 1 balance of " + b.sum_by['asset_alias'] + ": " + b.amount)
    })
  )
  // endsnippet

).then(() =>

  // snippet usd-iou-circulation
  client.balances.query({
    filter: 'asset_alias=$1',
    filter_params: ['bank1_usd_iou']
  }).then((results) =>
    results.items.forEach((b) => {
      console.log("Total circulation of Bank 1 USD IOU: " + b.amount)
    })
  )
  // endsnippet

).then(() =>

  // snippet account-balance-sum-by-currency
  client.balances.query({
    filter: 'account_alias=$1',
    filter_params: ['bank1'],
    sum_by: ['asset_definition.currency']
  }).then((results) =>
    results.items.forEach((b) => {
      var denom = b.sum_by['asset_definition.currency']
      console.log("Bank 1 balance of " + denom + "-denominated currencies: " + b.amount)
    })
  )
  // endsnippet

)
