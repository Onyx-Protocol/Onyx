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

  context 'issuance' do
    subject do
      tx = chain.transactions.build do |b|
        b.issue asset_alias: "gold-#{uuid}", amount: 100
        b.issue asset_alias: "silver-#{uuid}", amount: 200
        b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 100
        b.control_with_account account_alias: "bob-#{uuid}", asset_alias: "silver-#{uuid}", amount: 200
      end

      chain.transactions.submit(signer.sign(tx))
    end

    before { subject }

    it 'issues 100 units of gold to alice' do
      expect(account_balances("alice-#{uuid}")).to eq("gold-#{uuid}" => 100)
    end

    it 'issues 200 units of silver to bob' do
      expect(account_balances("bob-#{uuid}")).to eq("silver-#{uuid}" => 200)
    end
  end

  it 'fails to build transactions for non-existent assets' do
    expect {
      chain.transactions.build do |b|
        b.issue asset_alias: :unobtanium, amount: 100
      end
    }.to raise_error(Chain::APIError)
  end

  it 'fails to build an unbalanced transaction' do
    unbalanced = signer.sign(chain.transactions.build { |b|
      b.issue asset_alias: "gold-#{uuid}", amount: 1
      b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 100
    })

    expect { chain.transactions.submit(unbalanced) }.to raise_error(Chain::APIError)
  end

  context 'atomic swap' do
    before do
      issue(
        ["alice-#{uuid}", "gold-#{uuid}", 100],
        ["bob-#{uuid}", "silver-#{uuid}", 200]
      )

      swap_proposal = chain.transactions.build do |b|
        b.spend_from_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 10
        b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "silver-#{uuid}", amount: 20
      end

      swap_proposal = signer.sign(swap_proposal.allow_additional_actions)

      swap_tx = chain.transactions.build do |b|
        b.base_transaction swap_proposal
        b.spend_from_account account_alias: "bob-#{uuid}", asset_alias: "silver-#{uuid}", amount: 20
        b.control_with_account account_alias: "bob-#{uuid}", asset_alias: "gold-#{uuid}", amount: 10
      end

      chain.transactions.submit(signer.sign(swap_tx))
    end

    it 'gives alice 20 silver for 10 gold' do
      expect(
        account_balances("alice-#{uuid}")
      ).to eq("gold-#{uuid}" => 90, "silver-#{uuid}" => 20)
    end

    it 'gives bob 10 gold for 20 silver' do
      expect(
        account_balances("bob-#{uuid}")
      ).to eq("gold-#{uuid}" => 10, "silver-#{uuid}" => 180)
    end
  end
end
