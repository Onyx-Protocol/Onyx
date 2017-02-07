const chain = require('chain-sdk')
const client = new chain.Client()

const sleep = (ms) => new Promise(resolve => {
  setTimeout(() => resolve(), ms)
})

const resetCore = () => expect(
  client.config.reset()
    .then(() => ensureConfigured())
).to.be.fulfilled

const ensureConfigured = () => {
  const doConfig = () => client.config.info()
    .then((info) => {
      if (info.isConfigured) {
        return
      } else {
        return client.config.configure({ isGenerator: true })
          .then(() => sleep(1000))
      }
    })
    .catch(() => sleep(100).then(() => doConfig()))

  return expect(doConfig()).to.be.fulfilled
}

const setUpObjects = (signer) => {
  let keyResults, assetResults, accountResults
  let key

  return expect(Promise.all([
    client.mockHsm.keys.query({aliases: ['testkey']}),
    client.assets.query({filter: "alias='gold'"}),
    client.accounts.query({filter: "alias='alice'"}),
  ])).to.be.fulfilled
  .then((results) => {
    keyResults = results[0]
    assetResults = results[1]
    accountResults = results[2]

    let keyPromise = Promise.resolve()

    key = keyResults.items[0]
    if (!key) {
      keyPromise = keyPromise.then(() => expect(client.mockHsm.keys.create({alias: 'testkey'})
      .then((keyResp) => {
        key = keyResp
      }, (err) => {
        if (err.code == 'CH050') {
          return client.mockHsm.keys.query({aliases: ['testkey']}).then(resp => resp.items[0])
        }
        throw err
      })).to.be.fulfilled)
    }

    return keyPromise
  }).then(() => signer.addKey(key, client.mockHsm.signerConnection))
  .then(() => {
    const createPromises = []

    if (!assetResults.items[0]) createPromises.push(client.assets.create({alias: 'gold', rootXpubs: [key.xpub], quorum: 1}))
    if (!accountResults.items[0]) createPromises.push(client.accounts.create({alias: 'alice', rootXpubs: [key.xpub], quorum: 1}))

    return expect(Promise.all(createPromises)).to.be.fulfilled
  })
}

const issueTransaction = (signer) => expect(
  client.transactions.build((builder) => {
    builder.issue({ asset_alias: 'gold', amount: 100 })
    builder.controlWithAccount({ account_alias: 'alice', asset_alias: 'gold', amount: 100 })
  })
  .then(tpl => signer.sign(tpl))
  .then(tpl => client.transactions.submit(tpl))
).to.be.fulfilled

global.testHelpers = {
  sleep,
  resetCore,
  ensureConfigured,
  setUpObjects,
  issueTransaction,
}
