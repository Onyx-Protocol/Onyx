const uuid = require('uuid')

describe('mock hsm keys', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects()).to.be.fulfilled)
        .then(() => browser.url('/mockhsms'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all keys on the core', () => {
      browser.getText('.ItemList').should.contain('ALIAS')
      browser.getText('.ItemList').should.contain('XPUB')
      browser.getText('.ItemList').should.contain('testkey')
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('MockHSM keys')
      browser.getText('.PageTitle').should.contain('New MockHSM key')
    })
  })

  describe('creating keys', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/mockhsms'))
    )

    it('can create a new key', () => {
      const alias = 'test-key-' + uuid.v4()
      browser.waitForVisible('.ItemList button')
      browser.scroll('.ItemList button')
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.scroll('.FormContainer button')
      browser.click('.FormContainer button')
      browser.waitForVisible('.ItemList')
      browser.getText('.ItemList').should.contain('Created key. Create another?')
      browser.getText('.ItemList').should.contain(alias)
    })
  })

})
