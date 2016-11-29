const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
let asset_key, account_key

Promise.all([
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
]).then(keys => {
  asset_key = keys[0].xpub
  account_key = keys[1].xpub

  signer.addKey(asset_key, client.mockHsm.signerUrl)
  signer.addKey(account_key, client.mockHsm.signerUrl)
}).then(() => Promise.all([
  client.accounts.create({
    alias: 'acme_treasury',
    root_xpubs: [account_key],
    quorum: 1,
  }),

  // snippet create-asset-acme-common
  client.assets.create({
    alias: 'acme_common',
    root_xpubs: [asset_key],
    quorum: 1,
    tags: {
      internal_rating: '1',
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
      root_xpubs: [asset_key],
      quorum: 1,
      tags: {
        internal_rating: '2',
      },
      definition: {
        issuer: 'Acme Inc.',
        type: 'security',
        subtype: 'private',
        class: 'preferred',
    },
  })
  // endsnippet
])).then(() => {
  // snippet list-local-assets
  client.assets.query({
    filter: 'is_local=$1',
    filter_params: ['yes'],
  }).then(response => {
    for (let asset of response) {
      console.log('Local asset: ' + asset.alias)
    }
  })
  // endsnippet

  // snippet list-private-preferred-securities
  client.assets.query({
    filter: 'definition.type=$1 AND definition.subtype=$2 AND definition.class=$3',
    filter_params: ['security', 'private', 'preferred'],
  }).then(response => {
    for (let asset of response) {
      console.log('Private preferred security: ' + asset.alias)
    }
  })
  // endsnippet
}).then(() => {
  // snippet build-issue
  const issuePromise = client.transactions.build(function (builder) {
    builder.issue({
      asset_alias: 'acme_common',
      amount: 1000
    })
    builder.controlWithAccount({
      account_alias: 'acme_treasury',
      asset_alias: 'acme_common',
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
  const externalProgramPromise = client.accounts.createControlProgram({
    alias: 'acme_treasury',
  })

  return externalProgramPromise.then(externalProgram =>
    // snippet external-issue
    client.transactions.build(function (builder) {
      builder.issue({
        asset_alias: 'acme_preferred',
        amount: 2000
      })
      builder.controlWithProgram({
        control_program: externalProgram.control_program,
        asset_alias: 'acme_preferred',
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
  const retirePromise = client.transactions.build(function (builder) {
    builder.spendFromAccount({
      account_alias: 'acme_treasury',
      asset_alias: 'acme_common',
      amount: 50
    })
    builder.retire({
      asset_alias: 'acme_common',
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
}).then(() => {
  // snippet list-issuances
  client.transactions.query({
    filter: 'inputs(type=$1 AND asset_alias=$2)',
    filter_params: ['issue', 'acme_common'],
  }).then(response => {
    for (let tx of response) {
      console.log('Acme Common issued in tx ' + tx.id)
    }
  })
  // endsnippet

  // snippet list-transfers
  client.transactions.query({
    filter: 'inputs(type=$1 AND asset_alias=$2)',
    filter_params: ['spend', 'acme_common'],
  }).then(response => {
    for (let tx of response) {
      console.log('Acme Common transferred in tx ' + tx.id)
    }
  })
  // endsnippet

  // snippet list-retirements
  client.transactions.query({
    filter: 'outputs(type=$1 AND asset_alias=$2)',
    filter_params: ['retire', 'acme_common'],
  }).then(response => {
    for (let tx of response) {
      console.log('Acme Common retired in tx ' + tx.id)
    }
  })
  // endsnippet

  // snippet list-acme-common-balance
  client.balances.query({
    filter: 'asset_alias=$1',
    filter_params: ['acme_common'],
  }).then(response => {
    for (let balance of response) {
      console.log('Total circulation of Acme Common: ' + balance.amount)
    }
  })
  // endsnippet

  // snippet list-acme-balance
  client.balances.query({
    filter: 'asset_definition.issuer=$1',
    filter_params: ['Acme Inc.'],
  }).then(response => {
    for (let balance of response) {
      console.log('Total circulation of Acme stock ' + balance.sum_by['asset_alias'] + ':' + balance.amount)
    }
  })
  // endsnippet

  // snippet list-acme-common-unspents
  client.unspentOutputs.query({
    filter: 'asset_alias=$1',
    filter_params: ['acme_common'],
  }).then(response => {
    for (let unspent of response) {
      console.log('Acme Common held in output ' + unspent.transaction_id + ':' + unspent.position)
    }
  })
  // endsnippet
})
