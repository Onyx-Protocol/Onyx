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
      res[item.sumBy['assetAlias']] = item.amount
    })

    cb(null, res)
  }
}

describe('Callback style', () => {
  it('works', (done) => {
    const client = new chain.Client()
    const signer = new chain.HsmSigner()

    const aliceAlias = `alice-${uuid.v4()}`
    const bobAlias = `bob-${uuid.v4()}`
    const goldAlias = `gold-${uuid.v4()}`
    const silverAlias = `silver-${uuid.v4()}`
    const bronzeAlias = `bronze-${uuid.v4()}`
    const copperAlias = `copper-${uuid.v4()}`
    const issuancesAlias = `issuances-${uuid.v4()}`
    const spendsAlias = `spends-${uuid.v4()}`
    const tokenId = `token-${uuid.v4()}`

    let aliceKey, bobKey, goldKey, silverKey, otherKey, aliceId

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

      (next) => async.parallel([
        cb => client.assets.create({alias: goldAlias, rootXpubs: [goldKey.xpub], quorum: 1}, cb),
        cb => client.assets.create({alias: silverAlias, rootXpubs: [silverKey.xpub], quorum: 1}, cb)
      ], (err) => {
        expect(err).to.be.null
        next()
      }),

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
          builder.baseTransaction(swapProposal.rawTransaction)
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
          signer.signBatch(buildBatch.successes, cb)
          assert.equal(buildBatch.successes[1], null)
          assert.deepEqual([buildBatch.errors[0], buildBatch.errors[2], buildBatch.errors[3]], [null, null, null])
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

      // Control program creation

      (next) => client.accounts.createControlProgram({alias: aliceAlias}, (err, cp) => {
        assert(cp.controlProgram)
        next()
      }),
      (next) => client.accounts.createControlProgram({id: aliceId}, (err, cp) => {
        assert(cp.controlProgram)
        next()
      }),
      (next) => client.accounts.createControlProgram({}, (err) => {
        expect(err.code).to.equal('CH003')
        next()
      }),
      (next) => client.accounts.createControlProgram({alias: 'unobtalias'}, (err) => {
        expect(err.code).to.equal('CH002')
        next()
      }),

      // Pay to control program

      (next) => async.waterfall([
        cb => client.accounts.createControlProgram({alias: aliceAlias}, cb),
        (cp, cb) => client.transactions.build(builder => {
          builder.issue({
            assetAlias: goldAlias,
            amount: 1
          })
          builder.controlWithProgram({
            assetAlias: goldAlias,
            amount: 1,
            controlProgram: cp.controlProgram
          })
        }, cb),
        (issuance, cb) => signer.sign(issuance, cb),
        (signed, cb) => client.transactions.submit(signed, cb),
      ], (err, submitted) => {
        assert(submitted.id)
        next()
      }),

      // Transaction feeds

      (next) => async.parallel([
        cb => client.transactionFeeds.create({
          alias: issuancesAlias,
          filter: "inputs(type='issue')"
        }, cb),
        cb => client.transactionFeeds.create({
          alias: spendsAlias,
          filter: "inputs(type='spend')"
        }, cb)
      ], (err) => {
        expect(err).to.be.null
        next()
      }),

      () => done()
    ])
  })
})
