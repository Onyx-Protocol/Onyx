context 'access_tokens' do

  context 'creation' do
    let(:uid) { SecureRandom.uuid }
    subject(:token) { chain.access_tokens.create(id: uid) }

    it 'returns the created token' do
      expect(token.token).not_to be_empty
    end

    context 'after the token is created' do
      before { subject }

      it 'does not allow the same ID to be used twice' do
        expect {
          chain.access_tokens.create(type: :client, id: uid)
        }.to raise_error(Chain::APIError)
      end

      it 'token is returned in list' do
        tokens = chain.access_tokens.query().map(&:id)
        expect(tokens.include?(uid)).to eq(true)
      end

      it 'can delete the token' do
        chain.access_tokens.delete(uid)
        tokens = chain.access_tokens.query().map(&:id)
        expect(tokens.include?(uid)).to eq(false)
      end
    end
  end

  context 'deprecated syntax' do
    let(:uid1) { SecureRandom.uuid }
    let(:uid2) { SecureRandom.uuid }
    let(:client) { chain.access_tokens.create(type: :client, id: uid1) }
    let(:network) { chain.access_tokens.create(type: :network, id: uid2) }

    it 'creates client tokens' do
      expect(client.type).to eq('client')
    end

    it 'adds `client-readwrite` grant to client tokens' do
      client
      puts chain.authorization_grants.list_all
    end

    it 'creates network tokens' do
      expect(network.type).to eq('network')
    end

    it 'adds `crosscore` grant to network token'

    context 'deprecated roles' do
      before { client; network }

      it 'filters client tokens' do
        tokens = chain.access_tokens.query(type: :client).map(&:id)
        expect(tokens.include?(uid1)).to eq(true)
        expect(tokens.include?(uid2)).to eq(false)
      end

      it 'filters network tokens' do
        tokens = chain.access_tokens.query(type: :network).map(&:id)
        expect(tokens.include?(uid1)).to eq(false)
        expect(tokens.include?(uid2)).to eq(true)
      end
    end
  end
end