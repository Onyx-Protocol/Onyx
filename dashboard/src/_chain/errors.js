const lib = {
  create: function(type, message, props = {}) {
    let err
    if (props.body) {
      err = lib.newBatchError(props.body, props.requestId)
    } else {
      err = new Error(message)
    }

    err = Object.assign(err, props, {
      chainClientError: true,
      type: type,
    })
    return err
  },

  isChainError: function(err) {
    return err && !!err.chainClientError
  },

  isBatchError: function (v) {
    return v && v.code && !v.stack
  },

  newBatchError: function (body, requestId = false) {
    let err = new Error(lib.formatErrMsg(body, requestId))
    err.code = body.code
    err.chainMessage = body.message
    err.detail = body.detail
    err.requestId = requestId
    err.resp = body.resp
    return err
  },

  // TODO: remove me in favor of ErrorBanner.jsx rendering
  formatErrMsg: function(body, requestId) {
    let tokens = []

    if (typeof body.code === 'string' && body.code.length > 0) {
      tokens.push('Code: ' + body.code)
    }

    tokens.push('Message: ' + body.message)

    if (typeof body.detail === 'string' && body.detail.length > 0) {
      tokens.push('Detail: ' + body.detail)
    }

    if (requestId) {
      tokens.push('Request-ID: ' + requestId)
    }

    return tokens.join(' ')
  },

  types: {
    FETCH: 'FETCH',
    CONNECTIVITY: 'CONNECTIVITY',
    JSON: 'JSON',
    UNAUTHORIZED: 'UNAUTHORIZED',
    NOT_FOUND: 'NOT_FOUND',
    BAD_REQUEST: 'BAD_REQUEST',
    SERVER_ERROR: 'SERVER_ERROR',
  }
}

export default lib
