const shared = require('./shared')

module.exports = (client) => {
  return {
    create: (params) => shared.create(client, '/create-account', params),
    query: (params) => shared.query(client, '/list-accounts', params),
    createControlProgram: (opts = {}) => {
      const body = {type: 'account'}

      if (opts.alias) body.params = { account_alias: opts.alias }
      if (opts.id)    body.params = { account_id: opts.id }

      return shared.create(client, '/create-control-program', body)
    }
  }
}
