context 'accounts' do
  let(:key) { chain.mock_hsm.keys.create }
  let(:uuid) { SecureRandom.uuid }

  context 'creation' do
    subject { chain.accounts.create(alias: "alice-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }

    its(:id) { is_expected.not_to be_empty }

    it 'returns an error when required fields are missing' do
      expect { chain.accounts.create(alias: :fail) }.to raise_error(Chain::APIError)
    end
  end

  context 'batch creation' do
    subject {
      chain.accounts.create_batch([
        {alias: "carol-#{uuid}", root_xpubs: [key.xpub], quorum: 1}, # success
        {alias: "david-#{uuid}"}, # error
        {alias: "eve-#{uuid}", root_xpubs: [key.xpub], quorum: 1}, #success
      ])
    }

    its('successes.keys') { are_expected.to eq([0,2]) }
    its('errors.keys') { are_expected.to eq([1]) }

    it 'returns the reason for the error' do
      expect(subject.errors[1].code).to eq('CH202')
    end
  end

  context 'updating account tags' do
    let(:acc1) { chain.accounts.create(root_xpubs: [key.xpub], quorum: 1, tags: {x: 'one'}) }
    let(:acc2) { chain.accounts.create(root_xpubs: [key.xpub], quorum: 1, tags: {y: 'one'}) }
    let(:acc3) { chain.accounts.create(root_xpubs: [key.xpub], quorum: 1, tags: {z: 'one'}) }

    it 'updates individaul account tags' do
      chain.accounts.update_tags(id: acc1.id, tags: {x: 'two'})
      expect(
        chain.accounts.query(filter: "id='#{acc1.id}'").first.tags
      ).to eq('x' => 'two')
    end

    it 'returns an error when no id provided' do
      expect {
        chain.accounts.update_tags(tags: {x: 'three'})
      }.to raise_error(Chain::APIError)
    end

    context 'batch update' do
      subject {
        chain.accounts.update_tags_batch([
          {id: acc1.id, tags: {x: 'four'}},
          {tags: {y: 'four'}},
          {id: acc2.id, tags: {y: 'four'}},
          {id: acc3.id, alias: :redundant_alias, tags: {z: 'four'}},
        ])
      }

      its('successes.keys') { are_expected.to eq([0,2]) }
      its('errors.keys') { are_expected.to eq([1, 3]) }

      it 'returns an error for missing aliases' do
        expect(subject.errors[1].code).to eq('CH051')
      end

      it 'returns an error for redundant aliases' do
        expect(subject.errors[3].code).to eq('CH051')
      end

      it 'performs the update' do
        subject # perform batch request

        expect(
          chain.accounts.query(
            filter: "id=$1 OR id=$2",
            filter_params: [acc1.id, acc2.id]
          ).all.map(&:tags).reverse
        ).to eq([
          {'x' => 'four'},
          {'y' => 'four'},
        ])
      end
    end
  end
end
