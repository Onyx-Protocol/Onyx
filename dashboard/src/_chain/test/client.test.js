import assert from 'assert'
import nock from 'nock'

import Client from './client'
import errors from './errors'

function nockHelper() {
  return nock('http://test-server').defaultReplyHeaders({
    'Content-Type': 'application/json',
    'Chain-Request-Id': '1'
  })
}

describe('Client', () => {

  let client = new Client('http://test-server')

  describe('#request', () => {

    it('deserializes JSON responses', () => {
      nockHelper().post('/test').reply(200, '{"hello": "world"}')

      return client.request('/test').then((res) => {
        assert.deepEqual(res, {hello: 'world'})
      })
    })

    it('handles malformed URLs with FETCH error', () => {
      let client = new Client('not a URL')
      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.FETCH)
      })
    })

    it('handles unresponsive servers with FETCH error', () => {
      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.FETCH)
        assert(err.sourceError)
      })
    })

    it('handles missing request IDs with NO_REQUEST_ID error', () => {
      nock('http://test-server').post('/test').reply(200)

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.NO_REQUEST_ID)
        assert(err.response)
      })
    })

    it('handles bad JSON responses with JSON error', () => {
      nockHelper().post('/test').reply(200, 'not json')

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.JSON)
        assert(err.response)
      })
    })

    it('handles 401 status code with UNAUTHORIZED error', () => {
      nockHelper().post('/test').reply(401, '{}')

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.UNAUTHORIZED)
        assert(err.response)
      })
    })

    it('handles 404 status code with NOT_FOUND error', () => {
      nockHelper().post('/test').reply(404, '{}')

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.NOT_FOUND)
        assert(err.response)
      })
    })

    it('handles 4xx status code with BAD_REQUEST error', () => {
      nockHelper().post('/test').reply(400, '{}')

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.BAD_REQUEST)
        assert(err.response)
      })
    })

    it('handles 5xx status code with SERVER error', () => {
      nockHelper().post('/test').reply(500, '{}')

      return client.request('/test').then(() => {
        assert(false, 'should result in error')
      }).catch((err) => {
        assert.equal(err.type, errors.types.SERVER)
        assert(err.response)
      })
    })

  })

})
