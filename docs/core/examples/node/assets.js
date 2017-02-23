const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let assetKey, accountKey

Promise.all([
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
]).then(keys => {
  assetKey = keys[0].xpub
  accountKey = keys[1].xpub

  signer.addKey(assetKey, client.mockHsm.signerConnection)
  signer.addKey(accountKey, client.mockHsm.signerConnection)
}).then(() => Promise.all([
  client.accounts.create({
    alias: 'acme_treasury',
    rootXpubs: [accountKey],
    quorum: 1,
  }),

  // snippet create-asset-acme-common
  client.assets.create({
    alias: 'acme_common',
    rootXpubs: [assetKey],
    quorum: 1,
    tags: {
      internalRating: '1',
    },
    definition: {
      issuer: 'Acme Inc.',
      type: 'security',
      subtype: 'private',
      class: 'common',
    },
  })
  // endsnippet
  ,
  // snippet create-asset-acme-preferred
  client.assets.create({
    alias: 'acme_preferred',
    rootXpubs: [assetKey],
    quorum: 1,
    tags: {
      internalRating: '2',
    },
    definition: {
      issuer: 'Acme Inc.',
      type: 'security',
      subtype: 'private',
      class: 'preferred',
    },
  })
  // endsnippet
])).then(() =>
  // snippet list-local-assets
  client.assets.queryAll({
    filter: 'is_local=$1',
    filterParams: ['yes'],
  }, (asset, next) => {
    console.log('Local asset: ' + asset.alias)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-private-preferred-securities
  client.assets.queryAll({
    filter: 'definition.type=$1 AND definition.subtype=$2 AND definition.class=$3',
    filterParams: ['security', 'private', 'preferred'],
  }, (asset, next) => {
    console.log('Private preferred security: ' + asset.alias)
    next()
  })
  // endsnippet
).then(() => {
  // snippet build-issue
  const issuePromise = client.transactions.build(builder => {
    builder.issue({
      assetAlias: 'acme_common',
      amount: 1000
    })
    builder.controlWithAccount({
      accountAlias: 'acme_treasury',
      assetAlias: 'acme_common',
      amount: 1000
    })
  })
  // endsnippet

  return issuePromise.then(issueTx => {
    // snippet sign-issue
    const signingPromise = signer.sign(issueTx)
    // endsnippet

    return signingPromise
  }).then(signedIssueTx =>
    // snippet submit-issue
    client.transactions.submit(signedIssueTx)
    // endsnippet
  )
}).then(() => {
  const externalReceiverPromise = client.accounts.createReceiver({
    accountAlias: 'acme_treasury',
  })

  return externalReceiverPromise.then(externalReceiver =>
    // snippet external-issue
    client.transactions.build(builder => {
      builder.issue({
        assetAlias: 'acme_preferred',
        amount: 2000
      })
      builder.controlWithReceiver({
        receiver: externalReceiver,
        assetAlias: 'acme_preferred',
        amount: 2000
      })
    }).then(template => {
      return signer.sign(template)
    }).then(signed => {
      return client.transactions.submit(signed)
    })
    // endsnippet
  )
}).then(() => {
  // snippet build-retire
  const retirePromise = client.transactions.build(builder => {
    builder.spendFromAccount({
      accountAlias: 'acme_treasury',
      assetAlias: 'acme_common',
      amount: 50
    })
    builder.retire({
      assetAlias: 'acme_common',
      amount: 50
    })
  })
  // endsnippet

  return retirePromise.then(retireTx => {
    // snippet sign-retire
    const signingPromise = signer.sign(retireTx)
    // endsnippet

    return signingPromise
  }).then(signedRetireTx =>
    // snippet submit-retire
    client.transactions.submit(signedRetireTx)
    // endsnippet
  )
}).then(() =>
  // snippet list-issuances
  client.transactions.queryAll({
    filter: 'inputs(type=$1 AND asset_alias=$2)',
    filterParams: ['issue', 'acme_common'],
  }, (tx, next) => {
    console.log('Acme Common issued in tx ' + tx.id)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-transfers
  client.transactions.queryAll({
    filter: 'inputs(type=$1 AND asset_alias=$2)',
    filterParams: ['spend', 'acme_common'],
  }, (tx, next) => {
    console.log('Acme Common transferred in tx ' + tx.id)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-retirements
  client.transactions.queryAll({
    filter: 'outputs(type=$1 AND asset_alias=$2)',
    filterParams: ['retire', 'acme_common'],
  }, (tx, next) => {
    console.log('Acme Common retired in tx ' + tx.id)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-acme-common-balance
  client.balances.queryAll({
    filter: 'asset_alias=$1',
    filterParams: ['acme_common'],
  }, (balance, next) => {
    console.log('Total circulation of Acme Common: ' + balance.amount)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-acme-balance
  client.balances.queryAll({
    filter: 'asset_definition.issuer=$1',
    filterParams: ['Acme Inc.'],
  }, (balance, next) => {
    console.log('Total circulation of Acme stock ' + balance.sumBy.assetAlias + ': ' + balance.amount)
    next()
  })
  // endsnippet
).then(() =>
  // snippet list-acme-common-unspents
  client.unspentOutputs.queryAll({
    filter: 'asset_alias=$1',
    filterParams: ['acme_common'],
  }, (unspent, next) => {
    console.log('Acme Common held in output ' + unspent.id)
    next()
  })
  // endsnippet
).catch(err =>
  process.nextTick(() => { throw err })
)
