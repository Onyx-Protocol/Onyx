const uuid = require('uuid')

describe('access control', () => {
  before(() => {
    return expect(testHelpers.ensureConfigured()).to.be.fulfilled
  })

  describe('token list view', () => {
    before(() => {
      return browser.url('/access-control')
    })

    it('should redirect to /access-control?type=token', function() {
      browser.getUrl().should.contain('type=token')
    })

    it('displays the correct page', () => {
      browser.getText('.PageTitle').should.contain('Access control')
      browser.getText('thead').should.contain('TOKEN NAME')
    })

    it('shows a new token form ', () => {
      browser.waitForVisible('.AccessControlList .btn-primary')
      browser.scroll('.AccessControlList .btn-primary')
      browser.click('.AccessControlList .btn-primary')
      browser.waitForVisible('.FormContainer')
      browser.getUrl().should.contain('/access-control/create-token')
    })
  })
  describe('certificate list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/access-control?type=certificate'))
    })

    it('displays the correct page', () => {
      browser.getText('.PageTitle').should.contain('Access control')
      browser.getText('thead').should.contain('CERTIFICATE')
    })
  })

  describe('create a new access token', () => {
    beforeEach(() => {
      return browser.url('/access-control/create-token')
    })

    it('disables the submit button with no name', () => {
      browser.waitForVisible('button=Submit')
      browser.getAttribute('button=Submit', 'disabled').should.equal('true')
    })


    it('creates a token with no grants', () => {
      const name = 'test-token-' + uuid.v4()
      browser.setValue('input[name="guardData.id"]', name)
      browser.click('button=Submit')
      browser.waitForVisible('.Modal')
      browser.getText('.Modal').should.contain(name)
    })

    it('creates a token with all grants', () => {
      const name = 'test-token-' + uuid.v4()
      browser.setValue('input[name="guardData.id"]', name)
      browser.click('input[name="policies.client-readwrite"]')
      browser.click('input[name="policies.client-readonly"]')
      browser.click('input[name="policies.monitoring"]')
      browser.click('input[name="policies.crosscore"]')
      browser.click('input[name="policies.crosscore-signblock"]')
      browser.click('button=Submit')
      browser.waitForVisible('.Modal')
      browser.getText('.Modal').should.contain(name)
    })
  })


  describe('add a new certificate', () => {
    beforeEach(() => {
      return browser.url('/access-control/add-certificate')
    })

    // it('disables the submit button with no name', () => {
    //   browser.waitForVisible('button=Submit')
    //   browser.getAttribute('button=Submit', 'disabled').should.equal('true')
    // })
    //
    //
    // it('creates a token with no grants', () => {
    //   const name = 'test-token-' + uuid.v4()
    //   browser.setValue('input[name="guardData.id"]', name)
    //   browser.click('button=Submit')
    //   browser.waitForVisible('.Modal')
    //   browser.getText('.Modal').should.contain(name)
    // })
    //
    // it('creates a token with all grants', () => {
    //   const name = 'test-token-' + uuid.v4()
    //   browser.setValue('input[name="guardData.id"]', name)
    //   browser.click('input[name="policies.client-readwrite"]')
    //   browser.click('input[name="policies.client-readonly"]')
    //   browser.click('input[name="policies.monitoring"]')
    //   browser.click('input[name="policies.crosscore"]')
    //   browser.click('input[name="policies.crosscore-signblock"]')
    //   browser.click('button=Submit')
    //   browser.waitForVisible('.Modal')
    //   browser.getText('.Modal').should.contain(name)
    // })
  })
})
