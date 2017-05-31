describe 'TransactionFeed', nonparallel: true do
  let(:uuid) { SecureRandom.uuid }
  let(:produced_issuances) { [] }
  let(:produced_spends) { [] }
  let(:consumed_issuances) { [] }
  let(:consumed_spends) { [] }
  let(:key) { chain.mock_hsm.keys.create }

  let!(:gold) { chain.assets.create(alias: "gold-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:silver) { chain.assets.create(alias: "silver-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:alice) { chain.accounts.create(alias: "alice-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }
  let!(:bob) { chain.accounts.create(alias: "bob-#{uuid}", root_xpubs: [key.xpub], quorum: 1) }

  before do
    signer.add_key(key, chain.mock_hsm.signer_conn)

    chain.transaction_feeds.create(alias: "issuances-#{uuid}", filter: "inputs(type='issue')")
    chain.transaction_feeds.create(alias: "spends-#{uuid}", filter: "inputs(type='spend')")
  end

  context 'polling with ack-ing' do
    before do
      issuances_thread = Thread.new do
        f = chain.transaction_feeds.get(alias: "issuances-#{uuid}")
        f.consume do |tx|
          consumed_issuances << tx.id
          f.ack
          break if consumed_issuances.size == 2
        end
      end

      spends_thread = Thread.new do
        f = chain.transaction_feeds.get(alias: "spends-#{uuid}")
        f.consume do |tx|
          consumed_spends << tx.id
          f.ack
          break if consumed_spends.size == 2
        end
      end

      # Build issuances
      tx_batch = chain.transactions.build_batch([
        Chain::Transaction::Builder.new do |b|
          b.issue asset_alias: "gold-#{uuid}", amount: 1
          b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 1
        end,
        Chain::Transaction::Builder.new do |b|
          b.issue asset_alias: "silver-#{uuid}", amount: 1
          b.control_with_account account_alias: "bob-#{uuid}", asset_alias: "silver-#{uuid}", amount: 1
        end
      ]).successes.values
      submit_batch = chain.transactions.submit_batch(signer.sign_batch(tx_batch).successes.values)
      produced_issuances.concat(submit_batch.successes.values.map(&:id))

      # Build spends
      tx_batch = chain.transactions.build_batch([
        Chain::Transaction::Builder.new do |b|
          b.spend_from_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 1
          b.control_with_account account_alias: "bob-#{uuid}", asset_alias: "gold-#{uuid}", amount: 1
        end,
        Chain::Transaction::Builder.new do |b|
          b.spend_from_account account_alias: "bob-#{uuid}", asset_alias: "silver-#{uuid}", amount: 1
          b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "silver-#{uuid}", amount: 1
        end
      ]).successes.values
      submit_batch = chain.transactions.submit_batch(signer.sign_batch(tx_batch).successes.values)
      produced_spends.concat(submit_batch.successes.values.map(&:id))

      # Wait for tx feeds to exit threads
      issuances_thread.join
      spends_thread.join
    end

    it 'consumes every issuance transaction produced' do
      expect(consumed_issuances.sort).to eq(produced_issuances.sort)
    end

    it 'consumes every spend transaction produced' do
      expect(consumed_spends.sort).to eq(produced_spends.sort)
    end
  end

  context 'polling without ack-ing' do
    before do
      t = Thread.new do
        chain.transaction_feeds.get(alias: "issuances-#{uuid}").consume do |tx|
          consumed_issuances << tx.id
          break # NO ACK
        end
      end

      tx = chain.transactions.build do |b|
        b.issue asset_alias: "gold-#{uuid}", amount: 1
        b.control_with_account account_alias: "alice-#{uuid}", asset_alias: "gold-#{uuid}", amount: 1
      end
      produced_issuances << chain.transactions.submit(signer.sign(tx)).id

      t.join
    end

    it 'consumes the same transaction again' do
      consumed = nil
      t = Thread.new do
        chain.transaction_feeds.get(alias: "issuances-#{uuid}").consume do |tx|
          consumed = tx.id
          break # NO ACK
        end
      end
      t.join

      expect(consumed).to eq(consumed_issuances[0])
    end
  end

  context 'feed object management' do
    subject(:list) { chain.transaction_feeds.query.map(&:alias) }

    context 'query feeds' do
      it { is_expected.to include("spends-#{uuid}") }
      it { is_expected.to include("issuances-#{uuid}") }
    end

    context 'delete feeds' do
      before do
        chain.transaction_feeds.delete(alias: "issuances-#{uuid}")
      end

      it { is_expected.to include("spends-#{uuid}") }
      it { is_expected.not_to include("issuances-#{uuid}") }
    end
  end
end
