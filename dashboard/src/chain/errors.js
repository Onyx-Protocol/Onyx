export default {
  create: function(type, message, props = {}) {
    let err = new Error(message)
    Object.assign(err, props, {type: type})
    return err
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
