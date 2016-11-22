const shared = require('./shared')

module.exports = (client) => {
  return {
    create: (params) => shared.create(client, '/create-asset', params)
  }
}
