/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')
chai.use(chaiAsPromised)

const expect = chai.expect

// Helper function
const balanceByAssetAlias = (balances) => {
  let res = {}
  return Promise.resolve(balances)
  .then((balance) => {
    balance.items.forEach((item) => {
      res[item.sumBy.assetAlias] = item.amount
    })
    return res
  })
}

const client = new chain.Client()
const signer = new chain.HsmSigner()

const aliceAlias = `alice-${uuid.v4()}`
const bobAlias = `bob-${uuid.v4()}`
const goldAlias = `gold-${uuid.v4()}`
const silverAlias = `silver-${uuid.v4()}`
const bronzeAlias = `bronze-${uuid.v4()}`
const copperAlias = `copper-${uuid.v4()}`
const tokenId = `token-${uuid.v4()}`

let aliceKey, bobKey, goldKey, silverKey, otherKey, aliceId

const buildSignSubmit = buildFunc =>
  client.transactions.build(buildFunc)
    .then(tpl => signer.sign(tpl))
    .then(tpl => client.transactions.submit(tpl))

describe('Promise style', () => {
  before('set up API objects', () => {
    // Key creation and signer setup
    return expect(Promise.all([
      client.mockHsm.keys.create({alias: aliceAlias}),
      client.mockHsm.keys.create({alias: bobAlias}),
      client.mockHsm.keys.create({alias: goldAlias}),
      client.mockHsm.keys.create({alias: silverAlias}),
      client.mockHsm.keys.create({alias: bronzeAlias}),
      client.mockHsm.keys.create({alias: copperAlias}),
      client.mockHsm.keys.create(),
    ])).to.be.fulfilled

    .then(keys => {
      aliceKey = keys[0]
      bobKey = keys[1]
      goldKey = keys[2]
      silverKey = keys[3]
      otherKey = keys[6]

      signer.addKey(aliceKey, client.mockHsm.signerConnection)
      signer.addKey(bobKey, client.mockHsm.signerConnection)
      signer.addKey(goldKey, client.mockHsm.signerConnection)
      signer.addKey(silverKey, client.mockHsm.signerConnection)
    })

    // Account creation

    .then(() => expect(Promise.all([
      client.accounts.create({alias: aliceAlias, rootXpubs: [aliceKey.xpub], quorum: 1}),
      client.accounts.create({alias: bobAlias, rootXpubs: [bobKey.xpub], quorum: 1})
    ])).to.be.fulfilled)

    .then(accounts => {
      aliceId = accounts[0].id
    })

    .then(() =>
      expect(client.accounts.create({alias: 'david'}))
      // Request is missing key fields
      .to.be.rejectedWith('CH202')
    )

    // Asset creation

    .then(() => expect(Promise.all([
      client.assets.create({alias: goldAlias, rootXpubs: [goldKey.xpub], quorum: 1}),
      client.assets.create({alias: silverAlias, rootXpubs: [silverKey.xpub], quorum: 1})
    ])).to.be.fulfilled)

    .then(() =>
      expect(client.assets.create({alias: 'unobtanium'}))
      // Request is missing key fields
      .to.be.rejectedWith('CH202')
    )
  })

  it('works', () => {
    return Promise.resolve()

    // Access tokens

    .then(() =>
      expect(client.accessTokens.create({
        type: 'client',
        id: tokenId
      })).to.be.fulfilled
    )
    .then(resp => {
      expect(resp.token).to.not.be.empty
    })
    .then(() =>
      expect(client.accessTokens.create({
        type: 'client',
        id: tokenId
      }))
      // Using same ID twice will trigger a duplicate ID error
      .to.be.rejectedWith('CH302')
    )
    .then(() => expect(client.accessTokens.query()).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))

    .then(() => expect(client.accessTokens.query({type: 'client'})).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))

    .then(() => expect(client.accessTokens.query({type: 'network'})).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))

    .then(() => expect(client.accessTokens.delete(tokenId)).to.be.fulfilled )
    .then(() => expect(client.accessTokens.query()).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))

    // Batch account creation

    .then(() =>
      expect(client.accounts.createBatch([
        {alias: `carol-${uuid.v4()}`, rootXpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'david'},
        {alias: `eve-${uuid.v4()}`, rootXpubs: [otherKey.xpub], quorum: 1}, // success
      ])).to.be.fulfilled
    ).then(batchResponse => {
      assert.equal(batchResponse.successes[1], null)
      assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
    })

    // Batch asset creation

    .then(() =>
      expect(client.assets.createBatch([
        {alias: bronzeAlias, rootXpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'unobtanium'},
        {alias: copperAlias, rootXpubs: [otherKey.xpub], quorum: 1}, // success
      ])).to.be.fulfilled
    ).then(batchResponse => {
      assert.equal(batchResponse.successes[1], null)
      assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
    })

    // Account tag updates

    .then(() => {
      return expect(
        client.accounts.create({
          alias: `test-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        })
      ).to.be.fulfilled
    }).then(account => {
        return expect(
          client.accounts.updateTags({
            id: account.id,
            tags: {x: 1},
          }).then(() => {
            return account
          })
        ).to.be.fulfilled
    }).then(account => {
      return expect(
        client.accounts.query({
          filter: `id='${account.id}'`
        })
      ).to.be.fulfilled
    }).then(page => {
      assert.deepEqual(page.items[0].tags, {x: 1})
    })

    // Account tag update error

    .then(() => {
      return expect(
        client.accounts.updateTags({
          // ID intentionally omitted
          tags: {x: 1},
        })
      ).to.be.rejectedWith('CH051')
    })

    // Batch account tag updates

    .then(() => {
      return expect(
        client.accounts.createBatch([{
          alias: `x-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        }, {
          alias: `y-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {y: 0},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.accounts.updateTagsBatch([{
          id: batch.successes[0].id,
          tags: {x: 1},
        }, {
          id: batch.successes[1].id,
          tags: {y: 1},
        }]).then(() => {
          return batch
        })
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.accounts.query({
          filter: `id='${batch.successes[0].id}' OR id='${batch.successes[1].id}'`
        })
      ).to.be.fulfilled
    }).then(page => {
      assert.deepEqual(page.items.find(i => i.alias.match(/^x-/)).tags, {x: 1})
      assert.deepEqual(page.items.find(i => i.alias.match(/^y-/)).tags, {y: 1})
    })

    // Batch account tag update with errors

    .then(() => {
      return expect(
        client.accounts.createBatch([{
          alias: `x-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.accounts.updateTagsBatch([{
          id: batch.successes[0].id,
          tags: {x: 1},
        }, {
          // ID intentionally omitted
          tags: {y: 1},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      assert(batch.successes[0])
      assert(!batch.successes[1])
      assert(!batch.errors[0])
      assert(batch.errors[1])
    })

    // Asset tag updates

    .then(() => {
      return expect(
        client.assets.create({
          alias: `test-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        })
      ).to.be.fulfilled
    }).then(asset => {
        return expect(
          client.assets.updateTags({
            id: asset.id,
            tags: {x: 1},
          }).then(() => {
            return asset
          })
        ).to.be.fulfilled
    }).then(asset => {
      return expect(
        client.assets.query({
          filter: `id='${asset.id}'`
        })
      ).to.be.fulfilled
    }).then(page => {
      assert.deepEqual(page.items[0].tags, {x: 1})
    })

    // Asset tag update error

    .then(() => {
      return expect(
        client.assets.updateTags({
          // ID intentionally omitted
          tags: {x: 1},
        })
      ).to.be.rejectedWith('CH051')
    })

    // Batch asset tag updates

    .then(() => {
      return expect(
        client.assets.createBatch([{
          alias: `x-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        }, {
          alias: `y-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {y: 0},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.assets.updateTagsBatch([{
          id: batch.successes[0].id,
          tags: {x: 1},
        }, {
          id: batch.successes[1].id,
          tags: {y: 1},
        }]).then(() => {
          return batch
        })
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.assets.query({
          filter: `id='${batch.successes[0].id}' OR id='${batch.successes[1].id}'`
        })
      ).to.be.fulfilled
    }).then(page => {
      assert.deepEqual(page.items.find(i => i.alias.match(/^x-/)).tags, {x: 1})
      assert.deepEqual(page.items.find(i => i.alias.match(/^y-/)).tags, {y: 1})
    })

    // Batch asset tag update with errors

    .then(() => {
      return expect(
        client.assets.createBatch([{
          alias: `x-${uuid.v4()}`,
          rootXpubs: [otherKey.xpub],
          quorum: 1,
          tags: {x: 0},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      return expect(
        client.assets.updateTagsBatch([{
          id: batch.successes[0].id,
          tags: {x: 1},
        }, {
          // ID intentionally omitted
          tags: {y: 1},
        }])
      ).to.be.fulfilled
    }).then(batch => {
      assert(batch.successes[0])
      assert(!batch.successes[1])
      assert(!batch.errors[0])
      assert(batch.errors[1])
    })

    // Basic issuance

    .then(() =>
      expect(client.transactions.build(builder => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 100
        })
        builder.issue({
          assetAlias: silverAlias,
          amount: 200
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          accountAlias: bobAlias,
          assetAlias: silverAlias,
          amount: 200
        })
      })).to.be.fulfilled
      .then((issuance) => expect(signer.sign(issuance)).to.be.fulfilled)
      .then((signed) => expect(client.transactions.submit(signed)).to.be.fulfilled)
    )

    .then(() => expect(Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ])).to.be.fulfilled)
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 100})
      assert.deepEqual(balances[1], {[silverAlias]: 200})
    })

    // Bad singleton build call

    .then(() =>
      expect(client.transactions.build(builder => {
        builder.issue({
          assetAlias: 'unobtanium',
          amount: 100
        })
      }))
      // Non-existent asset
      .to.be.rejectedWith('CH002')
    )

    // Bad singleton submit call

    .then(() =>
      expect(client.transactions.build(builder => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 1
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
      })).to.be.fulfilled
      .then(issuance => expect(signer.sign(issuance)).to.be.fulfilled)
      .then(signed =>
        expect(client.transactions.submit(signed))
        // unbalanced transaction
        .to.be.rejectedWith('CH735')
      )
    )

    // Atomic swap

    .then(() =>
      expect(client.transactions.build(builder => {
        builder.spendFromAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 10
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: silverAlias,
          amount: 20
        })
      }))
      .to.be.fulfilled
      .then((swapProposal) => {
        swapProposal.allowAdditionalActions = true
        return expect(signer.sign(swapProposal)).to.be.fulfilled
      })
      .then((swapProposal) =>
        expect(client.transactions.build(builder => {
          builder.baseTransaction = swapProposal.rawTransaction
          builder.spendFromAccount({
            accountAlias: bobAlias,
            assetAlias: silverAlias,
            amount: 20
          })
          builder.controlWithAccount({
            accountAlias: bobAlias,
            assetAlias: goldAlias,
            amount: 10
          })
        }))
        .to.be.fulfilled)
        .then((swapTx) => expect(signer.sign(swapTx)).to.be.fulfilled)
        .then((signed) => expect(client.transactions.submit(signed)).to.be.fulfilled)
      )

    .then(() => Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ]))
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 90, [silverAlias]: 20})
      assert.deepEqual(balances[1], {[goldAlias]: 10, [silverAlias]: 180})
    })

    // Batch transactions

    .then(() => expect(client.transactions.buildBatch([
      // Should succeed
      (builder) => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
      },

      // Should fail at the build step
      (builder) => {
        builder.issue({
          assetAlias: 'foobar'
        })
      },

      // Should fail at the submit step
      (builder) => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 50
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
      },

      // Should succeed
      (builder) => {
        builder.issue({
          assetAlias: silverAlias,
          amount: 50
        })
        builder.controlWithAccount({
          accountAlias: bobAlias,
          assetAlias: silverAlias,
          amount: 50
        })
      }])).to.be.fulfilled
    )
    .then(buildBatch => {
      assert.equal(buildBatch.successes[1], null)
      assert.deepEqual([buildBatch.errors[0], buildBatch.errors[2], buildBatch.errors[3]], [null, null, null])
      return expect(signer.signBatch(buildBatch.successes)).to.be.fulfilled
    })
    .then(signedBatch => {
      assert(signedBatch.successes.indexOf(null) == -1)
      assert.deepEqual([signedBatch.errors[0], signedBatch.errors[1], signedBatch.errors[2]], [null, null, null])
      return expect(client.transactions.submitBatch(signedBatch.successes)).to.be.fulfilled
    })
    .then(submitBatch => {
      assert.equal(submitBatch.successes[1], null)
      assert.deepEqual([submitBatch.errors[0], submitBatch.errors[2]], [null, null])
    })

    // Receiver creation

    .then(() =>
      expect(client.accounts.createReceiver({accountAlias: aliceAlias})).to.be.fulfilled)
    .then(receiver => {
      assert(receiver.controlProgram)
      assert(receiver.expiresAt)
    })

    .then(() =>
      expect(client.accounts.createReceiver({accountId: aliceId})).to.be.fulfilled)
    .then(receiver => {
      assert(receiver.controlProgram)
      assert(receiver.expiresAt)
    })

    .then(() =>
      // Empty alias/id
      expect(client.accounts.createReceiver({}))
      .to.be.rejectedWith('CH002'))

    .then(() =>
      // Non-existent alias
      expect(client.accounts.createReceiver({accountAlias: 'unobtalias'}))
      .to.be.rejectedWith('CH002'))

    // Batch receiver creation

    .then(() =>
      expect(client.accounts.createReceiverBatch([
        {accountAlias: aliceAlias}, // success
        {accountAlias: 'unobtalias'},
        {accountAlias: bobAlias}, // success
      ])).to.be.fulfilled
    ).then(batchResponse => {
      assert.equal(batchResponse.successes[1], null)
      assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
    })

    // Pay to receiver

    .then(() =>
      expect(client.accounts.createReceiver({accountAlias: aliceAlias})).to.be.fulfilled)
    .then(receiver =>
      expect(client.transactions.build(builder => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 1,
        })
        builder.controlWithReceiver({
          receiver,
          assetAlias: goldAlias,
          amount: 1,
        })
      })).to.be.fulfilled)
    .then(issuance => expect(signer.sign(issuance)).to.be.fulfilled)
    .then(signed => expect(client.transactions.submit(signed)).to.be.fulfilled)
  })

  it('loads all results in `queryAll` requests', () => {
    let counter = 0
    return Promise.resolve()

    // Access tokens

    .then(() => expect(Promise.all([
      client.accessTokens.create({type: 'client', id: uuid.v4()}),
      client.accessTokens.create({type: 'client', id: uuid.v4()}),
    ])).to.be.fulfilled)
    .then(() => {
      counter = 0
      return expect(client.accessTokens.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Accounts

    .then(() => {
      counter = 0
      return expect(client.accounts.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Assets

    .then(() => {
      counter = 0
      return expect(client.assets.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // MockHsm keys

    .then(() => {
      counter = 0
      return expect(client.mockHsm.keys.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 6 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(6))

    // Transaction feeds

    .then(() => expect(Promise.all([
      client.transactionFeeds.create(),
      client.transactionFeeds.create(),
    ])).to.be.fulfilled)
    .then(() => {
      counter = 0
      return expect(client.transactionFeeds.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Transactions

    .then(() =>
      expect(client.transactions.buildBatch([builder => {
        builder.issue({ assetAlias: goldAlias, amount: 1 })
        builder.controlWithAccount({ accountAlias: aliceAlias, assetAlias: goldAlias, amount: 1 })
      }, builder => {
        builder.issue({ assetAlias: silverAlias, amount: 1 })
        builder.controlWithAccount({ accountAlias: bobAlias, assetAlias: silverAlias, amount: 1 })
      }])).to.be.fulfilled
      .then(issuanceBatch => expect(signer.signBatch(issuanceBatch.successes)).to.be.fulfilled)
      .then(signedBatch => expect(client.transactions.submitBatch(signedBatch.successes)).to.be.fulfilled)
    )

    .then(() => {
      counter = 0
      return expect(client.transactions.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Unspent Outputs

    .then(() => {
      counter = 0
      return expect(client.unspentOutputs.queryAll({pageSize: 1}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Balances

    .then(() => {
      counter = 0
      return expect(client.balances.queryAll({sumBy: ['asset_alias']}, (item, next, done) => {
        counter += 1
        expect(item).to.not.be.null
        counter >= 2 ? done() : next()
      })).to.be.fulfilled
    }).then(() => expect(counter).to.equal(2))

    // Rejection

    .then(() => expect(client.assets.queryAll({pageSize: 1}, (item, next, done) => {
      done(new Error('failure'))
    })).to.be.rejectedWith('failure'))
  })

  it('works with transaction feeds', () => {
    let issuanceFeed, spendFeed
    const feedIssuances = []
    const feedSpends = []
    const submittedIssuances = []
    const submittedSpends = []

    return Promise.resolve().then(() =>
      expect(client.transactionFeeds.create({
        filter: "inputs(type='issue')"
      })).to.be.fulfilled
    ).then(feed => {
      issuanceFeed = feed
    }).then(() =>
      expect(client.transactionFeeds.create({
        filter: "inputs(type='spend')"
      })).to.be.fulfilled
    ).then(feed => {
      spendFeed = feed
    }).then(() => {
      // Monitor feeds for issuances and spends, and then build/sign/submit
      // transactions that issue and spend.
      return expect(Promise.all([

        issuanceFeed.consume((tx, next, done) => {
          feedIssuances.push(tx.id)
          feedIssuances.length == 2 ? done(true) : next(true)
        }),

        spendFeed.consume((tx, next, done) => {
          feedSpends.push(tx.id)
          feedSpends.length == 2 ? done(true) : next(true)
        }),

        Promise.resolve().then(() =>
          buildSignSubmit(builder => {
            builder.issue({assetAlias: goldAlias, amount: 1})
            builder.controlWithAccount({accountAlias: aliceAlias, assetAlias: goldAlias, amount: 1})
          })
        ).then(tx =>
          submittedIssuances.push(tx.id)
        ).then(() =>
          buildSignSubmit(builder => {
            builder.spendFromAccount({accountAlias: aliceAlias, assetAlias: goldAlias, amount: 1})
            builder.controlWithAccount({accountAlias: bobAlias, assetAlias: goldAlias, amount: 1})
          })
        ).then(tx =>
          submittedSpends.push(tx.id)
        ).then(() =>
          buildSignSubmit(builder => {
            builder.issue({assetAlias: silverAlias, amount: 1})
            builder.controlWithAccount({accountAlias: bobAlias, assetAlias: silverAlias, amount: 1})
          })
        ).then(tx =>
          submittedIssuances.push(tx.id)
        ).then(() =>
          buildSignSubmit(builder => {
            builder.spendFromAccount({accountAlias: bobAlias, assetAlias: silverAlias, amount: 1})
            builder.controlWithAccount({accountAlias: aliceAlias, assetAlias: silverAlias, amount: 1})
          })
        ).then(tx =>
          submittedSpends.push(tx.id)
        )

      ])).to.be.fulfilled
    }).then(() => {
      assert.deepEqual(feedIssuances, submittedIssuances)
      assert.deepEqual(feedSpends, submittedSpends)
    })
  })

  describe('access control', () => {
    let tokenName
    let tokenGrant

    before('set up grant data', () => {
      tokenName = uuid.v4()
      return client.accessTokens.create({type: 'client', id: tokenName}).then(resp => {
        tokenGrant = {
          guard_data: { id: resp.id },
          guard_type: 'access_token',
          policy: 'client-readwrite'
        }
      })
    })

    it('can create access grants', () => {
      return Promise.resolve().then(() =>
        expect(client.authorizationGrants.create(tokenGrant)).to.be.fulfilled
      ).then(resp => {
        expect(resp.message == 'ok')
      })
    })

    it('can list access grants', () => {
      return Promise.resolve().then(() =>
        expect(client.authorizationGrants.create(tokenGrant)).to.be.fulfilled
      ).then(() =>
        expect(client.authorizationGrants.list()).to.be.fulfilled
      ).then(list => {
        let matched = false
        list.items.forEach((item) => {
          if (item.guardData.id == tokenName) {
            matched = true
          }
        })
        assert(matched)
      })
    })

    it('can delete access grants', () => {
      return Promise.resolve().then(() =>
        expect(client.authorizationGrants.create(tokenGrant)).to.be.fulfilled
      ).then(() =>
        expect(client.authorizationGrants.delete(tokenGrant)).to.be.fulfilled
      ).then(() =>
        expect(client.authorizationGrants.list()).to.be.fulfilled
      ).then(list => {
        let missing = true
        list.items.forEach((item) => {
          if (item.guardData.id == tokenName) {
            missing = false
          }
        })
        assert(missing)
      })
    })

    it('sanitizes X509 guard data', () =>
      expect(
        client.authorizationGrants.create({
          guardType: 'x509',
          guardData: {
            subject: {
              cn: tokenName,
              ou: 'test-ou',
            },
          },
          policy: 'client-readwrite',
        })
      ).to.be.fulfilled
      .then(g => {
        delete g.createdAt // ignore timestamp

        expect(g).deep.equals({
          guardType: 'x509',
          guardData: {
            subject: {
              cn: tokenName,
              ou: ['test-ou'],
            }
          },
          policy: 'client-readwrite',
          protected: false,
        })
      })
    )
  })
})
