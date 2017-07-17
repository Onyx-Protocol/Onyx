const uuid = require('uuid')

describe('assets', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects()).to.be.fulfilled)
        .then(() => browser.url('/assets'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all assets on the core', () => {
      browser.getText('.ItemList').should.contain('ASSET ALIAS')
      browser.getText('.ItemList').should.contain('gold')
      browser.getText('.ItemList').should.contain('View details')
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Assets')
      browser.getText('.PageTitle').should.contain('New asset')
    })
  })

  describe('creating assets', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/assets'))
    )

    it('can create a new asset', () => {
      const alias = 'test-asset-' + uuid.v4()
      browser.waitForVisible('.ItemList button')
      browser.scroll('.ItemList button')
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.scroll('.FormContainer input[type=radio][value=generate]')
      browser.click('.FormContainer input[type=radio][value=generate]')
      browser.scroll('.FormContainer button')
      browser.click('.FormContainer button')
      browser.waitForVisible('.AssetShow')
      browser.getText('.AssetShow').should.contain('Created asset. Create another?')
      browser.getText('.AssetShow').should.contain(alias)
    })
  })

  describe('showing Raw JSON', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAsset('asset-JSON-test'))
      .then((asset) => browser.url('/assets/' + asset.id))
    )

    it('can show Raw JSON of asset', () => {
      browser.waitForVisible('.AssetShow')
      browser.scroll('button=Raw JSON')
      browser.click('button=Raw JSON')
      browser.waitForVisible('.RawJsonModal')
      browser.getText('.RawJsonModal').should.contain('asset_pubkey')
    })
  })
})
