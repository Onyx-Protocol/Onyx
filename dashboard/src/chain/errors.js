const lib = {
  create: function(type, message, props = {}) {
    let err = new Error(message)
    err = {
      ...err,
      ...props.data,
      chainClientError: true,
      type: type,
    }
    return err
  },

  isChainError: function(err) {
    return err && !!err.chainClientError
  },

  isBatchError: function (v) {
    return v && v.code && !v.stack
  },

  newBatchError: function (opts) {
    let err = new Error(lib.formatErrMsg(opts))
    err.code = opts.code
    err.chainMessage = opts.message
    err.detail = opts.detail
    err.resp = opts.resp
    return err
  },

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
