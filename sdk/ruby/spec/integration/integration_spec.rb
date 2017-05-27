context 'Chain SDK integration test' do

  # TODO(jeffomatic): split this up with better organization.
  # This example assumes that a configured, empty core is hosted at localhost:1999.
  example 'integration test' do
    chain = Chain::Client.new
    signer = Chain::HSMSigner.new

    # Key creation and signer setup
    # TODO: remove me

    alice_key = chain.mock_hsm.keys.create(alias: :alice)
    signer.add_key(alice_key, chain.mock_hsm.signer_conn)

    bob_key = chain.mock_hsm.keys.create(alias: :bob)
    signer.add_key(bob_key, chain.mock_hsm.signer_conn)

    gold_key = chain.mock_hsm.keys.create(alias: :gold)
    signer.add_key(gold_key, chain.mock_hsm.signer_conn)

    silver_key = chain.mock_hsm.keys.create(alias: :silver)
    signer.add_key(silver_key, chain.mock_hsm.signer_conn)

    # Account creation
    # TODO: remove me

    alice = chain.accounts.create(alias: :alice, root_xpubs: [alice_key.xpub], quorum: 1)
    bob = chain.accounts.create(alias: :bob, root_xpubs: [bob_key.xpub], quorum: 1)

    # Asset creation
    # TODO: remove me

    chain.assets.create(alias: :gold, root_xpubs: [gold_key.xpub], quorum: 1)
    chain.assets.create(alias: :silver, root_xpubs: [silver_key.xpub], quorum: 1)

    # Receiver creation

    r = chain.accounts.create_receiver(account_alias: :alice)
    expect(r.control_program).not_to be_empty
    expect(r.expires_at).not_to be_nil

    r = chain.accounts.create_receiver(account_id: alice.id)
    expect(r.control_program).not_to be_empty
    expect(r.expires_at).not_to be_nil

    expect { chain.accounts.create_receiver({}) }.to raise_error(Chain::APIError)

    # Batch receiver creation

    receiver_batch = chain.accounts.create_receiver_batch([
      {account_alias: :alice}, # success
      {}, # error
      {account_id: alice.id}, #success
    ])

    expect(receiver_batch.errors.keys).to eq([1])
    expect(receiver_batch.successes.keys).to eq([0, 2])

    # Pay to receiver

    r = chain.accounts.create_receiver(account_alias: :alice)

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :gold, amount: 1
      b.control_with_receiver receiver: r, asset_alias: :gold, amount: 1
    end

    chain.transactions.submit(signer.sign(tx))

    # Transaction feed

    chain.transaction_feeds.create(alias: :issuances, filter: "inputs(type='issue')")
    chain.transaction_feeds.create(alias: :spends, filter: "inputs(type='spend')")

    consumed_issuances = []
    issuances_thread = Thread.new do
      f = chain.transaction_feeds.get(alias: :issuances)
      f.consume do |tx|
        consumed_issuances << tx.id
        f.ack
        break if consumed_issuances.size == 2
      end
    end

    consumed_spends = []
    spends_thread = Thread.new do
      f = chain.transaction_feeds.get(alias: :spends)
      f.consume do |tx|
        consumed_spends << tx.id
        f.ack
        break if consumed_spends.size == 2
      end
    end

    produced_issuances = []
    produced_spends = []

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :gold, amount: 1
      b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 1
    end
    produced_issuances << chain.transactions.submit(signer.sign(tx)).id

    tx = chain.transactions.build do |b|
      b.spend_from_account account_alias: :alice, asset_alias: :gold, amount: 1
      b.control_with_account account_alias: :bob, asset_alias: :gold, amount: 1
    end
    produced_spends << chain.transactions.submit(signer.sign(tx)).id

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :silver, amount: 1
      b.control_with_account account_alias: :bob, asset_alias: :silver, amount: 1
    end
    produced_issuances << chain.transactions.submit(signer.sign(tx)).id

    tx = chain.transactions.build do |b|
      b.spend_from_account account_alias: :bob, asset_alias: :silver, amount: 1
      b.control_with_account account_alias: :alice, asset_alias: :silver, amount: 1
    end
    produced_spends << chain.transactions.submit(signer.sign(tx)).id

    issuances_thread.join
    spends_thread.join

    expect(consumed_issuances).to eq(produced_issuances)
    expect(consumed_spends).to eq(produced_spends)

    # Test tx-feed ack behavior

    consumed = nil
    t = Thread.new do
      chain.transaction_feeds.get(alias: :issuances).consume do |tx|
        consumed = tx.id
        break # NO ACK
      end
    end

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :gold, amount: 1
      b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 1
    end
    produced = chain.transactions.submit(signer.sign(tx)).id

    t.join

    expect(consumed).to eq(produced)

    consumed = nil
    t = Thread.new do
      chain.transaction_feeds.get(alias: :issuances).consume do |tx|
        consumed = tx.id
        break # NO ACK
      end
    end

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :gold, amount: 1
      b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 1
    end
    chain.transactions.submit(signer.sign(tx)).id

    t.join

    expect(consumed).to eq(produced)

    # Query transaction feeds

    expect(
      chain.transaction_feeds.query.map(&:alias)
    ).to eq(['spends', 'issuances'])

    # Delete transaction feeds

    expect(chain.transaction_feeds.delete(alias: :issuances)).to be_nil

    expect(
      chain.transaction_feeds.query.map(&:alias)
    ).to eq(['spends'])
  end
end
