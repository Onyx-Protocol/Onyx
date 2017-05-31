context 'mock_hsm' do
  context 'creation' do
    let(:uuid) { SecureRandom.uuid }
    subject { chain.mock_hsm.keys.create(alias: uuid) }

    its(:xpub) { is_expected.not_to be_empty }

    context 'after the key is created' do
      before { subject }

      it 'does not allow the same ID to be used twice' do
        expect {
          chain.mock_hsm.keys.create(alias: uuid)
        }.to raise_error(Chain::APIError)
      end

      it 'key is returned in list' do
        keys = chain.mock_hsm.keys.query().map(&:alias)
        expect(keys).to include(uuid)
      end
    end
  end
end
