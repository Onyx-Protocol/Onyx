const chain = require('chain-sdk')
const uuid = require('uuid')

const client = new chain.Client()
let signer

describe('mock hsm keys', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      return expect(ensureConfigured()).to.be.fulfilled
        .then(() => expect(setUpObjects(signer)).to.be.fulfilled)
        .then(() => browser.url('/mockhsms'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.component-EmptyList').should.equal(false)
    })

    it('lists all assets on the core', () => {
      browser.getText('.component-ItemList').should.contain('ALIAS')
      browser.getText('.component-ItemList').should.contain('XPUB')
      browser.getText('.component-ItemList').should.contain('testkey')
    })

    it('displays the correct page title', () => {
      browser.getText('.component-PageTitle').should.contain('MockHSM keys')
      browser.getText('.component-PageTitle').should.contain('New MockHSM key')
    })
  })

  describe('creating assets', () => {
    before(() => expect(ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/mockhsms'))
    )

    it('can create a new asset', () => {
      const alias = 'test-key-' + uuid.v4()
      browser.click('.component-ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.click('.component-FormContainer button')
      browser.waitForVisible('.component-ItemList')
      browser.getText('.component-ItemList').should.contain('Created key. Create another?')
      browser.getText('.component-ItemList').should.contain(alias)
    })
  })

})
