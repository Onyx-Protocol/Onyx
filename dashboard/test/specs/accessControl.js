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

  describe('New access token form', () => {
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


  describe('New certificate form', () => {
    beforeEach(() => {
      return browser.url('/access-control/add-certificate')
    })

    it('returns an error for an empty form', () => {
      browser.waitForVisible('button=Submit')
      browser.click('button=Submit')
      browser.getText('.ErrorBanner').should.contain('Error submitting form')
    })

    it('returns an error with an empty subject', () => {
      browser.click('input[name="policies.client-readwrite"]')

      browser.waitForVisible('button=Submit')
      browser.click('button=Submit')
      browser.getText('.ErrorBanner').should.contain('X509 guard data contains invalid subject attribute')
    })

    it('returns an error with no policies', () => {
      const name = 'test-cert-' + uuid.v4()
      browser.selectByValue('.NewCertificateSubjectField:nth-child(1) select', 'cn')
      browser.setValue('.NewCertificateSubjectField:nth-child(1) input[type=text]', name)

      browser.waitForVisible('button=Submit')
      browser.click('button=Submit')
      browser.getText('.ErrorBanner').should.contain('You must specify one or more policies')
    })

    it('adds a new certificate', () => {
      const name = 'test-cert-' + uuid.v4()
      browser.selectByValue('.NewCertificateSubjectField:nth-child(1) select', 'cn')
      browser.setValue('.NewCertificateSubjectField:nth-child(1) input[type=text]', name)
      browser.click('button=Add Field')
      browser.selectByValue('.NewCertificateSubjectField:nth-child(2) select', 'o')
      browser.setValue('.NewCertificateSubjectField:nth-child(2) input[type=text]', name)
      browser.click('input[name="policies.client-readwrite"]')
      browser.click('button=Submit')
      browser.waitForVisible('.AccessControlList')
      browser.getText('.AccessControlList').should.contain('Granted policy to X509 certificate. Create another?')
      browser.getText('.AccessControlList').should.contain(`CN: ${name}`)
      browser.getText('.AccessControlList').should.contain(`O: ${name}`)
    })
  })
})
