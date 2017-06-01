/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()
const tokenId = `token-${uuid.v4()}`
const clientTokenId = `client-${uuid.v4()}`
const networkTokenId = `network-${uuid.v4()}`

describe('Access tokens test', () => {

  describe('Promise style', () => {

    describe('Access token creation', () => {

      it('access token creation successful', () => {
        return client.accessTokens.create({
          type: 'client',
          id: tokenId
        })
      })

      it('access token creation rejected due to duplicate ID', () => {
        return expect(client.accessTokens.create({
          type: 'client',
          id: tokenId
        }))
        .to.be.rejectedWith('CH302')
      })

      it('access token returned in list after creation', () => {
        return client.accessTokens.query()
          .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))
      })
    })

    describe('Access token deletion', () => {

      it('access token deletion successful', () => {
        return client.accessTokens.delete(tokenId)
      })

      it('access token deletion rejected due to missing ID', () => {
        return expect(client.accessTokens.delete())
          .to.be.rejectedWith('CH310')
      })

      it('access token removed from list after deletion', () => {
        return client.accessTokens.query()
          .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))
      })
    })

    describe('Deprecated syntax', () => {

      it('client token creation successful', () => {
        return client.accessTokens.create({
          type: 'client',
          id: clientTokenId
        }).then(resp => expect(resp.type).to.equal('client'))
      })

      it('client token creation adds client-readwrite grant', () => {
        return client.authorizationGrants.list()
          .then(resp => expect(resp.items.filter(guard => guard.guardData['id'] == clientTokenId)[0].policy).to.equal('client-readwrite'))
      })

      it('network token creation successful', () => {
        return client.accessTokens.create({
          type: 'network',
          id: networkTokenId
        }).then(resp => expect(resp.type).to.equal('network'))
      })

      it('network token creation adds crosscore grant', () => {
        return client.authorizationGrants.list()
          .then(resp => expect(resp.items.filter(guard => guard.guardData['id'] == networkTokenId)[0].policy).to.equal('crosscore'))
      })

      describe('Filtered client tokens', () => {

        it('contains clientTokenId', () => {
          return client.accessTokens.query({type: 'client'})
            .then(resp => expect(resp.items.map(item => item.id)).to.contain(clientTokenId))
        })

        it('does not contain networkTokenId', () => {
          return client.accessTokens.query({type: 'client'})
            .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(networkTokenId))
        })
      })

      describe('Filtered network tokens', () => {

        it('contains networkTokenId', () => {
          return client.accessTokens.query({type: 'network'})
            .then(resp => expect(resp.items.map(item => item.id)).to.contain(networkTokenId))
        })

        it('does not contain clientTokenId', () => {
          return client.accessTokens.query({type: 'network'})
            .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(clientTokenId))
        })
      })
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback style', () => {

    it('Access token creation', (done) => {
      client.accessTokens.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Access token deletion', (done) => {
      client.accessTokens.delete(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Access token query', (done) => {
      client.accessTokens.query(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })
  })
})
