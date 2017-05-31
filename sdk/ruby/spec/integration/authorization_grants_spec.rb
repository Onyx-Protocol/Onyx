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

    its(:guard_type) { is_expected.to eq('access_token') }
    its(:policy) { is_expected.to eq('client-readwrite') }

    it 'responds with the grant guard data id' do
      expect(subject.guard_data).to eq('id' => token.id)
    end

    context 'listing' do
      subject { chain.authorization_grants.list_all.find { |g|
        g.guard_data['id'] == token.id && !g.protected
      }}

      before { grant }

      it { is_expected.not_to be_nil }
      its(:guard_type) { is_expected.to eq('access_token') }
      its(:policy) { is_expected.to eq('client-readwrite') }
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

    its(:guard_type) { is_expected.to eq('x509') }
    its(:policy) { is_expected.to eq('crosscore') }

    it 'responds with the grant guard data common name' do
      expect(subject.guard_data['subject']['CN']).to eq("cn-#{uuid}")
    end

    context 'listing' do
      subject { chain.authorization_grants.list_all.find { |g|
        g.guard_type == 'x509' &&
        g.guard_data['subject']['CN'] == "cn-#{uuid}" &&
        !g.protected
      }}

      before { grant }

      it { is_expected.not_to be_nil }
      its(:policy) { is_expected.to eq('crosscore') }
    end
  end
end
