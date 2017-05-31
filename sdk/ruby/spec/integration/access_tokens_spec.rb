context 'access tokens' do
  context 'creation' do
    let(:uuid) { SecureRandom.uuid }
    subject(:token) { chain.access_tokens.create(id: uuid) }

    its(:token) { is_expected.not_to be_empty }

    context 'after the token is created' do
      before { subject }

      it 'does not allow the same ID to be used twice' do
        expect {
          chain.access_tokens.create(type: :client, id: uuid)
        }.to raise_error(Chain::APIError)
      end

      it 'token is returned in list' do
        tokens = chain.access_tokens.query().map(&:id)
        expect(tokens).to include(uuid)
      end

      it 'can delete the token' do
        chain.access_tokens.delete(uuid)
        tokens = chain.access_tokens.query().map(&:id)
        expect(tokens).not_to include(uuid)
      end
    end
  end

  context 'deprecated syntax' do
    let(:client_id) { SecureRandom.uuid }
    let(:network_id) { SecureRandom.uuid }
    let(:client) { chain.access_tokens.create(type: :client, id: client_id) }
    let(:network) { chain.access_tokens.create(type: :network, id: network_id) }

    it 'creates client tokens' do
      expect(client.type).to eq('client')
    end

    it 'adds `client-readwrite` grant to client tokens' do
      client
      grant = chain.authorization_grants.list_all
        .select{ |grant| grant.guard_data["id"] == client_id }
      expect(grant[0]).not_to be_nil
      expect(grant[0].policy).to eq('client-readwrite')
    end

    it 'creates network tokens' do
      expect(network.type).to eq('network')
    end

    it 'adds `crosscore` grant to network token' do
      network
      grant = chain.authorization_grants.list_all
        .select{ |grant| grant.guard_data["id"] == network_id }
      expect(grant[0]).not_to be_nil
      expect(grant[0].policy).to eq('crosscore')
    end

    context 'deprecated roles' do
      before { client; network }

      it 'filters client tokens' do
        tokens = chain.access_tokens.query(type: :client).map(&:id)
        expect(tokens).to include(client_id)
        expect(tokens).not_to include(network_id)
      end

      it 'filters network tokens' do
        tokens = chain.access_tokens.query(type: :network).map(&:id)
        expect(tokens).not_to include(client_id)
        expect(tokens).to include(network_id)
      end
    end
  end
end
