const chain = require('../dist/index.js')
const uuid = require('uuid')
const client = new chain.Client()
const signer = new chain.HsmSigner()

const balanceByAssetAlias = (balances) => {
  let res = {}
  return Promise.resolve(balances)
  .then((balance) => {
    balance.items.forEach((item) => {
      res[item.sumBy.assetAlias] = item.amount
    })
    return res
  })
}

const createAccount = (account = 'account') => {
  return client.mockHsm.keys.create()
    .then((key) => {
      signer.addKey(key, client.mockHsm.signerConnection)
      return client.accounts.create({
        alias: `${account}-${uuid.v4()}`,
        rootXpubs: [key.xpub],
        quorum: 1
      })
    })
}

const createAsset = (asset = 'asset') => {
  return client.mockHsm.keys.create()
    .then((key) => {
      signer.addKey(key, client.mockHsm.signerConnection)
      return client.assets.create({
        alias: `${asset}-${uuid.v4()}`,
        rootXpubs: [key.xpub],
        quorum: 1
      })
    })
}

const buildSignSubmit = (buildFunc, optClient, optSigner) => {
  const c = optClient || client
  const s = optSigner || signer
  return c.transactions.build(buildFunc)
    .then(tpl => s.sign(tpl))
    .then(tpl => c.transactions.submit(tpl))
}

module.exports = {
  balanceByAssetAlias,
  client,
  createAccount,
  createAsset,
  signer,
  buildSignSubmit,
}
