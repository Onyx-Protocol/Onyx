import 'isomorphic-fetch'
import errors from './errors'

class Client {
  constructor(baseUrl, clientToken) {
    this.baseUrl = baseUrl
    this.clientToken = clientToken
  }

  request(path, body = {}) {
    if (!body) {
      body = {}
    }

    let req = {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'User-Agent': 'chain-sdk-js/0.0'
      },
      body: JSON.stringify(body)
    }

    if (this.clientToken) {
      req.headers['Authorization'] = `Basic ${btoa(this.clientToken)}`
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
        return {status: 204}
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
          {response: resp, status: resp.status}
        )
      })
    })
  }
}

export default Client
