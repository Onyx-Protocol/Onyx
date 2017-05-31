context 'transactions' do
  let(:uuid) { SecureRandom.uuid }
  let(:key) { chain.mock_hsm.keys.create }

  let!(:gold) { chain.assets.create(alias: "gold-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:silver) { chain.assets.create(alias: "silver-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:alice) { chain.accounts.create(alias: "alice-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:bob) { chain.accounts.create(alias: "bob-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }

  before do
    signer.add_key(key, chain.mock_hsm.signer_conn)
  end

  let(:builders) do
    builders = []

    # Should succeed
    builders << Chain::Transaction::Builder.new do |b|
      b.issue asset_alias: "gold-#{uuid}", amount: 100
      b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 100
    end

    # Should fail at the build step
    builders << Chain::Transaction::Builder.new do |b|
      b.issue(asset_alias: :foobar)
    end

    # Should fail at the submit step
    builders << Chain::Transaction::Builder.new do |b|
      b.issue asset_alias: "gold-#{uuid}", amount: 50
      b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 100
    end

    # Should succeed
    builders << Chain::Transaction::Builder.new do |b|
      b.issue asset_alias: "silver-#{uuid}", amount: 100
      b.control_with_account account_alias: "bob-#{uuid}", asset_alias: "silver-#{uuid}", amount: 100
    end

    builders
  end

  context 'building' do
    subject { chain.transactions.build_batch(builders) }

    its('successes.keys') { are_expected.to eq([0, 2, 3]) }
    its('errors.keys') { are_expected.to eq([1]) }
  end

  context 'signing' do
    subject do
      built = chain.transactions.build_batch(builders)
      signer.sign_batch(built.successes.values)
    end

    its('successes.keys') { are_expected.to eq([0, 1, 2]) }
    its('errors.keys') { are_expected.to eq([]) }
  end

  context 'submitting' do
    subject do
      built = chain.transactions.build_batch(builders)
      signed = signer.sign_batch(built.successes.values)
      chain.transactions.submit_batch(signed.successes.values)
    end

    its('successes.keys') { are_expected.to eq([0, 2]) }
    its('errors.keys') { are_expected.to eq([1]) }
  end
end
