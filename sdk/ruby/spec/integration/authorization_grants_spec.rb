context 'authoriation grants' do
  let(:uuid) { SecureRandom.uuid }

  context 'token access' do
    let(:token) { chain.access_tokens.create(id: uuid) }

    subject(:grant) {
      chain.authorization_grants.create(
        guard_type: 'access_token',
        guard_data: {'id' => token.id},
        policy: 'client-readwrite'
      )
    }

    it 'responds with the grant guard type' do
      expect(subject.guard_type).to eq('access_token')
    end

    it 'responds with the grant guard data id' do
      expect(subject.guard_data).to eq('id' => token.id)
    end

    it 'responds with the grant policy' do
      expect(subject.policy).to eq('client-readwrite')
    end

    context 'listing' do
      subject { chain.authorization_grants.list_all.find { |g|
        g.guard_data['id'] == token.id && !g.protected
      }}

      before { grant }

      it 'finds the previously created grant' do
        expect(subject).not_to eq(nil)
      end

      it 'returns the an access token grant' do
        expect(subject.guard_type).to eq('access_token')
      end

      it 'returns the a client-readwrite grant' do
        expect(subject.policy).to eq('client-readwrite')
      end
    end

    context 'deletion' do
      subject { chain.authorization_grants.list_all }

      before do
        grant
        chain.authorization_grants.delete(
          guard_type: 'access_token',
          guard_data: {'id' => token.id},
          policy: 'client-readwrite'
        )
      end

      it 'no longer returns the deleted grant' do
        expect(subject.select { |g| g.guard_data['id'] == token.id }).to be_empty
      end
    end
  end

  context 'X509 access' do
    subject(:grant) {
      chain.authorization_grants.create(
        guard_type: 'x509',
        guard_data: {
          'subject' => {
            'CN' => "cn-#{uuid}",
            'OU' => "ou-#{uuid}",
          }
        },
        policy: 'crosscore'
      )
    }

    it 'responds with the grant guard type' do
      expect(subject.guard_type).to eq('x509')
    end

    it 'responds with the grant guard data common name' do
      expect(subject.guard_data['subject']['CN']).to eq("cn-#{uuid}")
    end

    it 'responds with the grant policy' do
      expect(subject.policy).to eq('crosscore')
    end

    context 'listing' do
      subject { chain.authorization_grants.list_all.find { |g|
        g.guard_type == 'x509' &&
        g.guard_data['subject']['CN'] == "cn-#{uuid}" &&
        !g.protected
      }}

      before { grant }

      it 'finds the previously created grant' do
        expect(subject).not_to eq(nil)
      end

      it 'returns the a client-readwrite grant' do
        expect(subject.policy).to eq('crosscore')
      end
    end
  end
end
