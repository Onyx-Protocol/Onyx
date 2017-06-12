/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()

function createToken() {
  return client.accessTokens.create({
    id: `token-${uuid.v4()}`
  })
}

describe('Access token', () => {

  it('creation successful', () => {
    return client.accessTokens.create({
      id: `another-${uuid.v4()}`
    }).then(resp => expect(resp.token).not.to.be.empty)
  })

  it('creation rejected due to duplicate ID', () => {
    return createToken()
      .then((token) => expect(client.accessTokens.create({
      id: token.id
    })).to.be.rejectedWith('CH302'))
  })

  it('returned in list after creation', () => {
    let tokenId
    return createToken()
      .then((token) => {
        tokenId = token.id
        return client.accessTokens.query()
      })
      .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))
  })

  it('deletion successful', () => {
    return createToken()
      .then((token) => client.accessTokens.delete(token.id))
      .then(resp => expect(resp.message).to.equal('ok'))
  })

  it('deletion rejected due to missing ID', () => {
    return createToken()
      .then(() => expect(client.accessTokens.delete())
      .to.be.rejectedWith('CH310'))
  })

  it('removed from list after deletion', () => {
    let tokenId
    return createToken()
      .then((token) => {
        tokenId = token.id
        return client.accessTokens.delete(tokenId)
      })
      .then(() => client.accessTokens.query())
      .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))
  })

  describe('queryAll', () => {
    it('success example', () => {
      let created
      const queried = []

      return createToken().then(token =>
        created = token.id
      ).then(() =>
        client.accessTokens.queryAll({}, (token, next, done) => {
          queried.push(token.id)
          next()
        })
      ).then(() =>
        expect(queried).to.include(created)
      )
    })
  })

  describe('Deprecated syntax', () => {
    let clientTokenId, networkTokenId

    describe('Client token', () => {

      it('creation successful', () => {
        return client.accessTokens.create({
          type: 'client',
          id: `client-${uuid.v4()}`
        }).then(resp => expect(resp.type).to.equal('client'))
      })

      it('creation adds client-readwrite grant', () => {
        return client.accessTokens.create({
          type: 'client',
          id: `client-${uuid.v4()}`
        }).then((resp) => {
          clientTokenId = resp.id
          return client.authorizationGrants.list()
        })
          .then(resp => expect(resp.items.filter(guard => guard.guardData['id'] == clientTokenId)[0].policy).to.equal('client-readwrite'))
      })
    })

    describe('Network token', () => {

      it('creation successful', () => {
        return client.accessTokens.create({
          type: 'network',
          id: `network-${uuid.v4()}`
        }).then(resp => expect(resp.type).to.equal('network'))
      })

      it('creation adds crosscore grant', () => {
        return client.accessTokens.create({
          type: 'network',
          id: `network-${uuid.v4()}`
        }).then((resp) => {
          networkTokenId = resp.id
          return client.authorizationGrants.list()
        })
          .then(resp => expect(resp.items.filter(guard => guard.guardData['id'] == networkTokenId)[0].policy).to.equal('crosscore'))
      })
    })

    describe('Deprecated roles', () => {

      beforeEach(() => {
        clientTokenId = `client-${uuid.v4()}`
        networkTokenId = `network-${uuid.v4()}`

        return client.accessTokens.create({
          type: 'client',
          id: clientTokenId
        }).then(() => client.accessTokens.create({
          type: 'network',
          id: networkTokenId
        }))
      })

      it('filtered by client type contains clientTokenId', () => {
        return client.accessTokens.query({type: 'client'})
          .then(resp => expect(resp.items.map(item => item.id)).to.contain(clientTokenId))
      })

      it('filtered by client type does not contain networkTokenId', () => {
        return client.accessTokens.query({type: 'client'})
          .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(networkTokenId))
      })

      it('filtered by network type contains networkTokenId', () => {
        return client.accessTokens.query({type: 'network'})
          .then(resp => expect(resp.items.map(item => item.id)).to.contain(networkTokenId))
      })

      it('filtered by network type does not contain clientTokenId', () => {
        return client.accessTokens.query({type: 'network'})
          .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(clientTokenId))
      })
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('create', (done) => {
      client.accessTokens.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('delete', (done) => {
      client.accessTokens.delete(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('query', done => {
      client.accessTokens.query(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('queryAll', done => {
      client.accessTokens.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })

  })
})
