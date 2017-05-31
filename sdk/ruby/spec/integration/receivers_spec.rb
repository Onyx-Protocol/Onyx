describe 'Receiver' do
  let(:uuid) { SecureRandom.uuid }
  let(:key) { chain.mock_hsm.keys.create }
  let(:account_alias) { "receiver-account-#{uuid}" }

  let!(:account) { chain.accounts.create(alias: account_alias, root_xpubs: [key.xpub], quorum: 1) }

  subject(:accounts) { chain.accounts }

  describe 'create a new receiver' do
    let(:params) {{}}
    subject { accounts.create_receiver(params)}

    it 'raises an error with no params' do
      expect { subject }.to raise_error(Chain::APIError)
    end

    context 'by account alias' do
      let(:params) { {account_alias: account_alias} }

      its(:control_program) { is_expected.not_to be_empty }
      its(:expires_at) { is_expected.not_to be_nil }
      its(:expires_at) { is_expected.to be }
    end

    context 'by account id' do
      let(:params) { {account_id: account.id} }

      its(:control_program) { is_expected.not_to be_empty }
      its(:expires_at) { is_expected.not_to be_nil }
    end

    context 'with an expiration time' do
      let(:params) { {account_id: account.id} }
    end
  end

  describe 'create multiple new receivers' do
    let(:params) {[
      {account_alias: account_alias}, # success
      {}, # error
      {account_id: account.id}, #success
    ]}
    subject { accounts.create_receiver_batch(params) }

    its('errors.keys') { is_expected.to eq([1]) }
    its('successes.keys') { is_expected.to eq([0, 2]) }
  end

  describe 'pay to a receiver' do
    let(:asset) { chain.assets.create(root_xpubs: [key.xpub], quorum: 1) }
    let(:receiver) { chain.accounts.create_receiver(account_alias: account_alias) }

    before do
      signer.add_key(key, chain.mock_hsm.signer_conn)
      tx = chain.transactions.build do |b|
        b.issue asset_id: asset.id, amount: 1
        b.control_with_receiver receiver: receiver, asset_id: asset.id, amount: 1
      end

      chain.transactions.submit(signer.sign(tx))
    end

    subject { account_balances(account_alias) }

    it 'pays assets to the receiving account' do
      expect(subject[asset.id]).to eq(1)
    end
  end
end
