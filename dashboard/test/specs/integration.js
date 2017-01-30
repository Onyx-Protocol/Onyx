const chain = require('chain-sdk')

const client = new chain.Client()
let signer

const sleep = (ms) => {
  let current
  const start = new Date()

  do {
    current = new Date()
  } while((current - start) < ms)
}

const setUpObjects = () => {
  let keyResults, assetResults, accountResults
  let key
  signer = new chain.HsmSigner()

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
      })).to.be.fulfilled)
    }

    return keyPromise.then(() => signer.addKey(key, client.mockHsm.signerConnection))
  }).then(() => {
    const createPromises = []

    if (!assetResults.items[0]) createPromises.push(client.assets.create({alias: 'gold', rootXpubs: [key.xpub], quorum: 1}))
    if (!accountResults.items[0]) createPromises.push(client.accounts.create({alias: 'alice', rootXpubs: [key.xpub], quorum: 1}))

    return expect(Promise.all(createPromises)).to.be.fulfilled
  })
}

const issueTransaction = () => expect(
  client.transactions.build((builder) => {
    builder.issue({ asset_alias: 'gold', amount: 100 })
    builder.controlWithAccount({ account_alias: 'alice', asset_alias: 'gold', amount: 100 })
  })
  .then(tpl => signer.sign(tpl))
  .then(tpl => client.transactions.submit(tpl))
).to.be.fulfilled

describe('dashboard', () => {
  describe('homepage', () => {
    before(() => browser.url('/'))

    it('should load the page', function() {
      browser.getTitle().should.equal('Chain Core Dashboard')
    })

    it('should redirect to /transactions', function() {
      browser.getUrl().should.contain('/transactions')
    })
  })

  describe('transaction list', () => {
    before(() =>
      expect(setUpObjects()).to.be.fulfilled
      .then(() => expect(issueTransaction()).to.be.fulfilled)
      .then(() => browser.url('/transactions')))

    it('lists all blockchain transactions', () => {
      browser.getText('.component-ItemList').should.contain('alice')
      browser.getText('.component-ItemList').should.contain('gold')
      browser.getText('.component-ItemList').should.contain('100')
    })

    it('displays the correct page title', () => {
      browser.getText('.component-PageTitle').should.contain('Transactions')
      browser.getText('.component-PageTitle').should.contain('New transaction')
    })
  })

  describe('empty states', () => {
    before(() =>
      expect(client.config.reset()
        .then(() => {
          sleep(1000)
          return client.config.configure({ isGenerator: true })
        })
      ).to.be.fulfilled
      .then(() => browser.url('/transactions'))
    )

    it('displays a welcome message', () => {
      browser.getText('.component-EmptyList').should.contain('Welcome to Chain Core')
      browser.getText('.component-EmptyList').should.contain('New transaction')
    })
  })
})
