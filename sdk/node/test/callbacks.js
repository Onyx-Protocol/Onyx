/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const async = require('async')
const assert = require('assert')
const chai = require('chai')
const expect = chai.expect

// Helper function
const balanceByAssetAlias = (cb) => {
  return (err, balances) => {
    let res = {}

    balances.items.forEach((item) => {
      res[item.sumBy.assetAlias] = item.amount
    })

    cb(null, res)
  }
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

describe('Callback style', () => {
  before('set up API objects', (done) => {
    async.series([
      // Key creation and signer setup

      (next) => async.parallel([
        cb => client.mockHsm.keys.create({alias: aliceAlias}, cb),
        cb => client.mockHsm.keys.create({alias: bobAlias}, cb),
        cb => client.mockHsm.keys.create({alias: goldAlias}, cb),
        cb => client.mockHsm.keys.create({alias: silverAlias}, cb),
        cb => client.mockHsm.keys.create({alias: bronzeAlias}, cb),
        cb => client.mockHsm.keys.create({alias: copperAlias}, cb),
        cb => client.mockHsm.keys.create({}, cb)
      ], (err, keys) => {
        expect(keys.length).to.equal(7)

        aliceKey = keys[0]
        bobKey = keys[1]
        goldKey = keys[2]
        silverKey = keys[3]
        otherKey = keys[6]

        signer.addKey(aliceKey, client.mockHsm.signerConnection)
        signer.addKey(bobKey, client.mockHsm.signerConnection)
        signer.addKey(goldKey, client.mockHsm.signerConnection)
        signer.addKey(silverKey, client.mockHsm.signerConnection)

        next()
      }),

      // Account creation

      (next) => async.parallel([
        cb => client.accounts.create({alias: aliceAlias, rootXpubs: [aliceKey.xpub], quorum: 1}, cb),
        cb => client.accounts.create({alias: bobAlias, rootXpubs: [bobKey.xpub], quorum: 1}, cb)
      ], (err, accounts) => {
        expect(err).to.be.null
        aliceId = accounts[0].id
        next()
      }),

      (next) => client.accounts.create({alias: 'david'}, (err) => {
        // Request is missing key fields
        expect(err.code).to.equal('CH202')
        next()
      }),

      // Asset creation

      (next) => async.parallel([
        cb => client.assets.create({alias: goldAlias, rootXpubs: [goldKey.xpub], quorum: 1}, cb),
        cb => client.assets.create({alias: silverAlias, rootXpubs: [silverKey.xpub], quorum: 1}, cb)
      ], (err) => {
        expect(err).to.be.null
        next()
      }),

      () => done()
    ])
  })

  it('works', (done) => {

    async.series([
      // Access tokens

      (next) => client.accessTokens.create({type: 'client', id: tokenId}, (err, resp) => {
        expect(resp.token).to.not.be.empty
        expect(err).to.be.null
        next()
      }),

      (next) => client.accessTokens.create({type: 'client', id: tokenId}, (err, resp) => {
        expect(resp).to.be.null
        expect(err).to.not.be.null
        expect(err.code).to.equal('CH302')
        next()
      }),

      (next) => client.accessTokens.query({}, (err, resp) => {
        expect(err).to.be.null
        expect(resp.items.map(item => item.id)).to.contain(tokenId)
        next()
      }),

      (next) => client.accessTokens.query({type: 'client'}, (err, resp) => {
        expect(resp.items.map(item => item.id)).to.contain(tokenId)
        next()
      }),

      (next) => client.accessTokens.query({type: 'network'}, (err, resp) => {
        expect(resp.items.map(item => item.id)).to.not.contain(tokenId)
        next()
      }),

      (next) => client.accessTokens.delete(tokenId, (err, resp) => {
        expect(resp.message).to.equal('ok')
        next()
      }),

      (next) => client.accessTokens.query({}, (err, resp) => {
        expect(resp.items.map(item => item.id)).to.not.contain(tokenId)
        next()
      }),

      // Batch account creation

      (next) => client.accounts.createBatch([
        {alias: `carol-${uuid.v4()}`, rootXpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'david'},
        {alias: `eve-${uuid.v4()}`, rootXpubs: [otherKey.xpub], quorum: 1}, // success
      ], (err, batchResponse) => {
        assert.equal(batchResponse.successes[1], null)
        assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
        next()
      }),

      // Asset creation

      (next) => client.assets.create({alias: 'unobtanium'}, (err) => {
        // Request is missing key fields
        expect(err.code).to.equal('CH202')
        next()
      }),

      // Batch asset creation

      (next) => client.assets.createBatch([
        {alias: 'unobtanium'},
        {alias: bronzeAlias, rootXpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: copperAlias, rootXpubs: [otherKey.xpub], quorum: 1}, // success
      ], (err, batchResponse) => {
        assert.equal(batchResponse.successes[0], null)
        assert.deepEqual([batchResponse.errors[1], batchResponse.errors[2]], [null, null])
        next()
      }),

      // Account tag updates
      // This just tests that the callback is engaged correctly. Behavior is
      // tested in the promises test.

      (next) => client.accounts.updateTags(
        {}, // intentionally blank
        err => {
          assert(err)
          next()
        }
      ),

      (next) => client.accounts.updateTagsBatch(
        [{}, {}], // intentionally blank
        (err, batch) => {
          assert(!err)
          assert(batch.errors[0])
          assert(batch.errors[1])
          next()
        }
      ),

      // Asset tag updates
      // This just tests that the callback is engaged correctly. Behavior is
      // tested in the promises test.

      (next) => client.assets.updateTags(
        {}, // intentionally blank
        err => {
          assert(err)
          next()
        }
      ),

      (next) => client.assets.updateTagsBatch(
        [{}, {}], // intentionally blank
        (err, batch) => {
          assert(!err)
          assert(batch.errors[0])
          assert(batch.errors[1])
          next()
        }
      ),

      // Basic issuance

      (next) => async.waterfall([
        cb => client.transactions.build(builder => {
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
        }, cb),

        (issuance, cb) => signer.sign(issuance, cb),
        (signed, cb) => client.transactions.submit(signed, cb),
      ], (err, result) => {
        expect(err).to.be.null
        expect(result.id).to.not.be.blank
        next()
      }),

      (next) => async.parallel([
        cb => client.balances.query({filter: `account_alias='${aliceAlias}'`}, balanceByAssetAlias(cb)),
        cb => client.balances.query({filter: `account_alias='${bobAlias}'`}, balanceByAssetAlias(cb))
      ], (err, balances) => {
        expect(err).to.be.null
        assert.deepEqual(balances[0], {[goldAlias]: 100})
        assert.deepEqual(balances[1], {[silverAlias]: 200})
        next()
      }),

      // Bad singleton build call

      (next) => client.transactions.build(builder => {
        builder.issue({
          assetAlias: 'unobtanium',
          amount: 100
        })
      }, (err) => {
        // Non-existent asset
        expect(err.code).to.equal('CH002')
        next()
      }),

      // Bad singleton submit call

      (next) => async.waterfall([
        cb => client.transactions.build(builder => {
          builder.issue({
            assetAlias: goldAlias,
            amount: 1
          })
          builder.controlWithAccount({
            accountAlias: aliceAlias,
            assetAlias: goldAlias,
            amount: 100
          })
        }, cb),

        (issuance, cb) => signer.sign(issuance, cb),
        (signed, cb) => client.transactions.submit(signed, cb)
      ], (err) => {
        expect(err.code).to.equal('CH735')
        next()
      }),

      // Atomic swap

      (next) => async.waterfall([
        cb => client.transactions.build(builder => {
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
        }, cb),

        (swapProposal, cb) => {
          swapProposal.allowAdditionalActions = true
          signer.sign(swapProposal, cb)
        },

        (swapProposal, cb) => client.transactions.build(builder => {
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
        }, cb),

        (swapTx, cb) => signer.sign(swapTx, cb),
        (signed, cb) => client.transactions.submit(signed, cb)
      ], (err, result) => {
        expect(err).to.be.null
        expect(result.id).to.not.be.blank
        next()
      }),

      (next) => async.parallel([
        cb => client.balances.query({filter: `account_alias='${aliceAlias}'`}, balanceByAssetAlias(cb)),
        cb => client.balances.query({filter: `account_alias='${bobAlias}'`}, balanceByAssetAlias(cb))
      ], (err, balances) => {
        expect(err).to.be.null
        assert.deepEqual(balances[0], {[goldAlias]: 90, [silverAlias]: 20})
        assert.deepEqual(balances[1], {[goldAlias]: 10, [silverAlias]: 180})
        next()
      }),


      // Batch transactions
      (next) => async.waterfall([
        cb => client.transactions.buildBatch([
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
            builder.issue({assetAlias: 'foobar' })
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
          }
        ], cb),
        (buildBatch, cb) => {
          assert.equal(buildBatch.successes[1], null)
          assert.deepEqual([buildBatch.errors[0], buildBatch.errors[2], buildBatch.errors[3]], [null, null, null])
          signer.signBatch(buildBatch.successes, cb)
        },
        (signedBatch, cb) => {
          assert(signedBatch.successes.indexOf(null) == -1)
          assert.deepEqual([signedBatch.errors[0], signedBatch.errors[1], signedBatch.errors[2]], [null, null, null])
          client.transactions.submitBatch(signedBatch.successes, cb)
        }
      ], (err, submitBatch) => {
        assert.equal(submitBatch.successes[1], null)
        assert.deepEqual([submitBatch.errors[0], submitBatch.errors[2]], [null, null])
        next()
      }),

      // Receiver creation

      (next) => client.accounts.createReceiver({accountAlias: aliceAlias}, (err, cp) => {
        assert(cp.controlProgram)
        assert(cp.expiresAt)
        next()
      }),
      (next) => client.accounts.createReceiver({accountId: aliceId}, (err, cp) => {
        assert(cp.controlProgram)
        assert(cp.expiresAt)
        next()
      }),
      (next) => client.accounts.createReceiver({}, (err) => {
        expect(err.code).to.equal('CH002')
        next()
      }),
      (next) => client.accounts.createReceiver({accountAlias: 'unobtalias'}, (err) => {
        expect(err.code).to.equal('CH002')
        next()
      }),

      // Batch receiver creation

      (next) => client.accounts.createReceiverBatch([
          {accountAlias: aliceAlias}, // success
          {accountAlias: 'unobtalias'},
          {accountAlias: bobAlias}, // success
      ], (err, batchResponse) => {
        assert.equal(batchResponse.successes[1], null)
        assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
        next()
      }),

      // Pay to receiver

      (next) => async.waterfall([
        cb => client.accounts.createReceiver({accountAlias: aliceAlias}, cb),
        (receiver, cb) => client.transactions.build(builder => {
          builder.issue({
            assetAlias: goldAlias,
            amount: 1
          })
          builder.controlWithReceiver({
            receiver,
            assetAlias: goldAlias,
            amount: 1,
          })
        }, cb),
        (issuance, cb) => signer.sign(issuance, cb),
        (signed, cb) => client.transactions.submit(signed, cb),
      ], (err, submitted) => {
        assert(err === null)
        assert(submitted.id)
        next()
      }),

      () => done()
    ])
  })

  it('loads all results in `queryAll` requests', (done) => {
    let counter = 0

    async.series([

      (nextStep) => async.parallel([
        cb => client.accessTokens.create({type: 'client', id: uuid.v4()}, cb),
        cb => client.accessTokens.create({type: 'client', id: uuid.v4()}, cb),
      ], () => {
        counter = 0
        client.accessTokens.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      }),

      // Accounts

      (nextStep) => {
        counter = 0
        client.accounts.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      },

      // Assets

      (nextStep) => {
        counter = 0
        client.assets.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      },

      // MockHsm keys

      (nextStep) => {
        counter = 0
        client.mockHsm.keys.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      },

      // Transaction feeds

      (nextStep) => async.parallel([
        cb => client.transactionFeeds.create({}, cb),
        cb => client.transactionFeeds.create({}, cb),
      ], () => {
        counter = 0
        client.transactionFeeds.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      }),

      // Transactions

      (nextStep) => async.waterfall([
        cb => client.transactions.buildBatch([
          (builder) => {
            builder.issue({ assetAlias: goldAlias, amount: 100 })
            builder.controlWithAccount({ accountAlias: aliceAlias, assetAlias: goldAlias, amount: 100 })
          },
          (builder) => {
            builder.issue({ assetAlias: silverAlias, amount: 100 })
            builder.controlWithAccount({ accountAlias: bobAlias, assetAlias: silverAlias, amount: 100 })
          }
        ], cb),
        (buildBatch, cb) => signer.signBatch(buildBatch.successes, cb),
        (signedBatch, cb) => client.transactions.submitBatch(signedBatch.successes, cb)
      ], () => {
        counter = 0
        client.transactions.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      }),

      // Unspent Outputs

      (nextStep) => {
        counter = 0
        client.unspentOutputs.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      },

      // Balances

      (nextStep) => {
        counter = 0
        client.balances.queryAll({pageSize: 1}, (item, next, done) => {
          counter += 1
          expect(item).to.not.be.null
          counter >= 2 ? done() : next()
        }, () => {
          expect(counter).to.equal(2)
          nextStep()
        })
      },

      // Rejection

      (nextStep) => {
        counter = 0
        client.assets.queryAll({pageSize: 1}, (item, next, done) => {
          done(new Error('failure'))
        }, (err) => {
          expect(err.message).to.equal('failure')
          nextStep()
        })
      },

      () => done()

    ])
  })

  it('works with transaction feeds', (done) => {
    let issuanceFeed, spendFeed
    const feedIssuances = []
    const feedSpends = []
    let submittedIssuances = []
    let submittedSpends = []

    async.series([
      (next) => client.transactionFeeds.create({
        filter: "inputs(type='issue')"
      }, (err, feed) => {
        issuanceFeed = feed
        expect(err).to.be.null
        next()
      }),

      (next) => client.transactionFeeds.create({
        filter: "inputs(type='spend')"
      }, (err, feed) => {
        spendFeed = feed
        expect(err).to.be.null
        next()
      }),

      (next) => async.parallel([
        // Feed consumers
        cb => issuanceFeed.consume((tx, next, done) => {
          feedIssuances.push(tx.id)
          feedIssuances.length == 2 ? done(true) : next(true)
        }, () => { cb() }),
        cb => spendFeed.consume((tx, next, done) => {
          feedSpends.push(tx.id)
          feedSpends.length == 2 ? done(true) : next(true)
        }, () => { cb() }),

        // Transactions to read

        cb => async.waterfall([
          cb2 => client.transactions.buildBatch([
            (builder) => {
              builder.issue({assetAlias: goldAlias, amount: 1})
              builder.controlWithAccount({accountAlias: aliceAlias, assetAlias: goldAlias, amount: 1})
            },
            (builder) => {
              builder.issue({assetAlias: silverAlias, amount: 1})
              builder.controlWithAccount({accountAlias: bobAlias, assetAlias: silverAlias, amount: 1})
            }
          ], cb2),
          (buildBatch, cb2) => signer.signBatch(buildBatch.successes, cb2),
          (signedBatch, cb2) => client.transactions.submitBatch(signedBatch.successes,
            (err, submittedBatch) => {
              submittedIssuances = submittedBatch.successes.map((tx) => tx.id)
              cb2()
            }
          ),
          (cb2) => {
            client.transactions.buildBatch([
              (builder) => {
                builder.spendFromAccount({accountAlias: aliceAlias, assetAlias: goldAlias, amount: 1})
                builder.controlWithAccount({accountAlias: bobAlias, assetAlias: goldAlias, amount: 1})
              },
              (builder) => {
                builder.spendFromAccount({accountAlias: bobAlias, assetAlias: silverAlias, amount: 1})
                builder.controlWithAccount({accountAlias: aliceAlias, assetAlias: silverAlias, amount: 1})
              }
            ], cb2)
          },
          (buildBatch, cb2) => signer.signBatch(buildBatch.successes, cb2),
          (signedBatch, cb2) => client.transactions.submitBatch(signedBatch.successes,
            (err, submittedBatch) => {
              submittedSpends = submittedBatch.successes.map((tx) => tx.id)
              cb2()
            }
          ),

          () => cb()
        ])
      ], () => {
        assert.deepEqual(feedIssuances.sort(), submittedIssuances.sort())
        assert.deepEqual(feedSpends.sort(), submittedSpends.sort())
        next()
      }),

      () => done()
    ])
  })

  describe('access control', () => {
    let tokenName
    let tokenGrant

    before('set up grant data', (done) => {
      tokenName = uuid.v4()
      async.series([
        (next) => client.accessTokens.create({type: 'client', id: tokenName}, (err, resp) => {
          tokenGrant = {
            guard_data: { id: resp.id },
            guard_type: 'access_token',
            policy: 'client-readwrite'
          }
          next()
        }),
        () => done()
      ])
    })

    it('can create access grants', (done) => {
      client.authorizationGrants.create(tokenGrant, (err, resp) => {
        expect(resp.message == 'ok')
        done()
      })
    })

    it('can list access grants', (done) => {
      async.series([
        (next) => client.authorizationGrants.create(tokenGrant, () => next()),
        (next) => client.authorizationGrants.list((err, list) => {
            let matched = false
            list.items.forEach((item) => {
              if (item.guardData.id == tokenName) {
                matched = true
              }
            })
            assert(matched)
            done()
        })
      ])
    })


    it('can delete access grants', (done) => {
      async.series([
        (next) => client.authorizationGrants.create(tokenGrant, () => next()),
        (next) => client.authorizationGrants.delete(tokenGrant, () => next()),
        (next) => client.authorizationGrants.list((err, list) => {
          let missing = true
          list.items.forEach((item) => {
            if (item.guardData.id == tokenName) {
              missing = false
            }
          })
          assert(missing)
          done()
        })
      ])
    })
  })
})
