const shared = require('./shared')

module.exports = (client) => {
  return {
    query: (params) => shared.query(client, '/list-balances', params),
  }
}
