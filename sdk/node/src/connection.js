// FIXME: Microsoft Edge has issues returning errors for responses
// with a 401 status. We should add browser detection to only
// use the ponyfill for unsupported browsers.
const { fetch } = require('fetch-ponyfill')()
const errors = require('./errors')
const btoa = require('btoa')

const blacklistAttributes = [
  'after',
  'asset_tags',
  'asset_definition',
  'account_tags',
  'next',
  'reference_data',
  'tags',
]

const snakeize = (object) => {
  for(let key in object) {
    let value = object[key]
    let newKey = key

    if (/[A-Z]/.test(key)) {
      newKey = key.replace(/([A-Z])/g, v => `_${v.toLowerCase()}`)
      delete object[key]
    }

    if (typeof value == 'object' && !blacklistAttributes.includes(key)) {
      value = snakeize(value)
    }

    object[newKey] = value
  }

  return object
}

const camelize = (object) => {
  for (let key in object) {
    let value = object[key]
    let newKey = key

    if (/_/.test(key)) {
      newKey = key.replace(/([_][a-z])/g, v => v[1].toUpperCase())
      delete object[key]
    }

    if (typeof value == 'object' && !blacklistAttributes.includes(key)) {
      value = camelize(value)
    }

    object[newKey] = value
  }

  return object
}

/**
 * @class
 * Connection information for an instance of Chain Core.
 */
class Connection {
  /**
   * constructor - create a new Chain client object capable of interacting with
   * the specified Chain Core.
   *
   * @param {string} baseUrl Chain Core URL.
   * @param {string} token   Chain Core client token for API access.
   * @returns {Client}
   */
  constructor(baseUrl, token) {
    this.baseUrl = baseUrl || 'http://localhost:1999'
    this.token = token || ''
  }

  /**
   * Submit a request to the specified Chain Core.
   *
   * @param  {string} path
   * @param  {object} [body={}]
   * @return {Promise}
   */
  request(path, body = {}) {
    if (!body) {
      body = {}
    }

    // Convert camelcased request body field names to use snakecase for API
    // processing.
    const snakeBody = snakeize(body) // Ssssssssssss

    let req = {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',

        // TODO(jeffomatic): The Fetch API has inconsistent behavior between
        // browser implementations and polyfills.
        //
        // - For Edge: we can't use the browser's fetch API because it doesn't
        // always returns a WWW-Authenticate challenge to 401s.
        // - For Safari/Chrome: using fetch-ponyfill (the polyfill) causes
        // console warnings if the user agent string is provided.
        //
        // For now, let's not send the UA string.
        //'User-Agent': 'chain-sdk-js/0.0'
      },
      body: JSON.stringify(snakeBody)
    }

    if (this.token) {
      req.headers['Authorization'] = `Basic ${btoa(this.token)}`
    }

    return fetch(this.baseUrl + path, req).catch((err) => {
      throw errors.create(
        errors.types.FETCH,
        'Fetch error: ' + err.toString(),
        {sourceError: err}
      )
    }).then((resp) => {
      let requestId = resp.headers.get('Chain-Request-Id')
      if (!requestId) {
        throw errors.create(
          errors.types.NO_REQUEST_ID,
          'Chain-Request-Id header is missing. There may be an issue with your proxy or network configuration.',
          {response: resp}
        )
      }

      if (resp.status == 204) {
        return { status: 204 }
      }

      return resp.json().catch(() => {
        throw errors.create(
          errors.types.JSON,
          'Could not parse JSON response',
          {response: resp, status: resp.status}
        )
      }).then((body) => {
        if (resp.status / 100 == 2) {
          return body
        }

        // Everything else is a status error.
        let errType = null
        if (resp.status == 401) {
          errType = errors.types.UNAUTHORIZED
        } else if (resp.status == 404) {
          errType = errors.types.NOT_FOUND
        } else if (resp.status / 100 == 4) {
          errType = errors.types.BAD_REQUEST
        } else {
          errType = errors.types.SERVER
        }

        throw errors.create(
          errType,
          errors.formatErrMsg(body, requestId),
          {
            response: resp,
            status: resp.status,
            body: body,
            requestId: requestId
          }
        )
      }).then((body) => {
        // After processing the response, convert snakecased field names to
        // camelcase to match language conventions.
        return camelize(body)
      })
    })
  }
}

module.exports = Connection
