const uuid = require('uuid')

describe('accounts', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects()).to.be.fulfilled)
        .then(() => browser.url('/accounts'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all accounts on the core', () => {
      browser.elements('.ListItem').value.length.should.be.above(0)
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Accounts')
      browser.getText('.PageTitle').should.contain('New account')
    })
  })

  describe('creating accounts', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/accounts'))
    )

    it('can create a new account', () => {
      const alias = 'test-account-' + uuid.v4()
      browser.waitForVisible('.ItemList button')
      browser.scroll('.ItemList button')
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.scroll('.FormContainer input[type=radio][value=generate]')
      browser.click('.FormContainer input[type=radio][value=generate]')
      browser.scroll('.FormContainer button')
      browser.click('.FormContainer button')
      browser.waitForVisible('.AccountShow')
      browser.getText('.AccountShow').should.contain('Created account. Create another?')
      browser.getText('.AccountShow').should.contain(alias)
    })
  })

  describe('updating accounts', () => {
    let updateAccount
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAccount('account-update-test'))
      .then((account) => updateAccount = account)
    )

    it('can update account tags', () => {
      browser.url('/accounts/' + updateAccount.id)
      browser.waitForVisible('.AccountShow')
      browser.scroll('.glyphicon-pencil')
      browser.click('.glyphicon-pencil')
      browser.waitForVisible('.FormContainer')
      browser.getText('.PageTitle').should.contain('Edit account tags')
      browser.setValue('textarea', '{"updated": true')
      browser.scroll('button=Submit')
      browser.click('button=Submit')
      browser.waitForVisible('.AccountShow')
      browser.getText('.AccountShow').should.contain('{"updated": true')
    })
  })

  describe('creating receivers', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAccount('receiver-test'))
      .then((account) => browser.url('/accounts/' + account.id))
    )

    it('can create a new receiver', () => {
      browser.waitForVisible('.AccountShow')
      browser.scroll('button=Create receiver')
      browser.click('button=Create receiver')
      browser.waitForVisible('.ReceiverModal')
      browser.getText('.ReceiverModal').should.contain('Copy this one-time use receiver to use in a transaction')
    })
  })

  describe('showing Raw JSON', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAccount('account-JSON-test'))
      .then((account) => browser.url('/accounts/' + account.id))
    )

    it('can show Raw JSON of account', () => {
      browser.waitForVisible('.AccountShow')
      browser.scroll('button=Raw JSON')
      browser.click('button=Raw JSON')
      browser.waitForVisible('.RawJsonModal')
      browser.getText('.RawJsonModal').should.contain('account_xpub')
    })
  })

  describe('filtering', () => {
    const tag1 = uuid.v4()
        , tag2 = uuid.v4()
        , id1 = uuid.v4()
        , id2 = uuid.v4()
        , id3 = uuid.v4()

    before(() => {
      expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAccount(id1, {tags: {x: tag1}}))
      .then(() => testHelpers.createAccount(id2, {tags: {x: tag2}}))
      .then(() => testHelpers.createAccount(id3, {tags: {x: tag1}}))
      .then(() => browser.url('/accounts'))
    })

    it('filters properly', () => {
      browser.waitForVisible(".SearchBar form input[type='search']")
      browser.setValue(".SearchBar form input[type='search']", `tags.x='${tag1}'`)
      browser.submitForm('.SearchBar form')
      browser.waitForVisible('.ListItem')

      browser.elements('.ListItem').value.length.should.equal(2)
      browser.elements('.ListItem').value[0].getText().should.contain(id3)
      browser.elements('.ListItem').value[1].getText().should.contain(id1)
    })
  })
})
