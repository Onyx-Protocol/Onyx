/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()
const xAssetAlias = `x-${uuid.v4()}`
const yAssetAlias = `y-${uuid.v4()}`

let mockHsmKey

describe('Asset', () => {

  before('set up API objects', () => {

    // Key and asset creation
    return client.mockHsm.keys.create()
      .then(key => { mockHsmKey = key })
      .then(() => {
        return client.assets.create({alias: xAssetAlias, rootXpubs: [mockHsmKey.xpub], quorum: 1, tags: {x: 0}})
      })
      .then(() => {
        return client.assets.create({alias: yAssetAlias, rootXpubs: [mockHsmKey.xpub], quorum: 1, tags: {y: 0}})
      })
  })

  describe('Single asset creation', () => {

    it('successful', () => {
      return client.assets.create({alias: `asset-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1})
        .then(resp => expect(resp.id).not.to.be.empty)
    })

    it('rejected due to missing key fields', () => {
      return expect(client.assets.create({alias: 'asset'})).to.be.rejectedWith('CH202')
    })
  })

  describe('Batch asset creation', () => {
    let batchResponse = {}

    before(() => client.assets.createBatch([
        {alias: `bronze-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1}, // success
        {alias: 'unobtanium'}, // failure
        {alias: `copper-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1}, // success
      ])
      .then(resp => {batchResponse = resp})
    )

    it('returns two successes', () => assert.equal(batchResponse.successes[1], null))
    it('returns one error', () => assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null]))
  })

  describe('Single asset tags update', () => {

    it('successful', () => {
      return client.assets.updateTags({
        alias: xAssetAlias,
        tags: {x: 1},
      })
      .then(() => {
        return client.assets.query({
          filter: `alias='${xAssetAlias}'`
        })
      })
      .then(page => {
        assert.deepEqual(page.items[0].tags, {x: 1})
      })
    })

    it('rejected due to missing ID/Alias', () => {
      return expect(
        client.assets.updateTags({
          // ID/Alias intentionally omitted
          tags: {x: 1},
        })
      ).to.be.rejectedWith('CH051')
    })
  })

  describe('Batch asset tags update', () => {

    it('successful', () => {
      return client.assets.updateTagsBatch([{
        alias: xAssetAlias,
        tags: {x: 2},
      }, {
        alias: yAssetAlias,
        tags: {y: 2},
      }])
      .then(() => {
        return client.assets.query({
          filter: `alias='${xAssetAlias}' OR alias='${yAssetAlias}'`
        })
      })
      .then(page => {
        assert.deepEqual(page.items.find(i => i.alias.match(/^x-/)).tags, {x: 2})
        assert.deepEqual(page.items.find(i => i.alias.match(/^y-/)).tags, {y: 2})
      })
    })

    it('fails to update assets tags with missing ID/Alias', () => {
      return client.assets.updateTagsBatch([{
        alias: xAssetAlias,
        tags: {x: 3},
      }, {
        // ID/Alias intentionally omitted
        tags: {y: 3}
      }])
      .then(batch => {
        assert(batch.successes[0])
        assert(!batch.successes[1])
        assert(!batch.errors[0])
        assert(batch.errors[1])
      })
    })
  })

  describe('queryAll', () => {
    it('success example', () => {
      let created
      const queried = []

      return client.assets.create({
        rootXpubs: [mockHsmKey.xpub],
        quorum: 1
      }).then(asset =>
        created = asset.id
      ).then(() =>
        client.assets.queryAll({}, (asset, next, done) => {
          queried.push(asset.id)
          next()
        })
      ).then(() => {
        expect(queried).to.include(created)
      })
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('create', (done) => {
      client.assets.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('createBatch', (done) => {
      client.assets.createBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('updateTags', (done) => {
      client.assets.updateTags(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('updateTagsBatch', (done) => {
      client.assets.updateTagsBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('query', done => {
      client.assets.query({}, done)
    })

    it('queryAll', done => {
      client.assets.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })

  })
})
