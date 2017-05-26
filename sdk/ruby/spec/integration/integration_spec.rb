context 'Chain SDK integration test' do

  # TODO(jeffomatic): split this up with better organization.
  # This example assumes that a configured, empty core is hosted at localhost:1999.
  example 'integration test' do
    chain = Chain::Client.new
    signer = Chain::HSMSigner.new

    # Key creation and signer setup

    alice_key = chain.mock_hsm.keys.create(alias: :alice)
    signer.add_key(alice_key, chain.mock_hsm.signer_conn)

    bob_key = chain.mock_hsm.keys.create(alias: :bob)
    signer.add_key(bob_key, chain.mock_hsm.signer_conn)

    gold_key = chain.mock_hsm.keys.create(alias: :gold)
    signer.add_key(gold_key, chain.mock_hsm.signer_conn)

    silver_key = chain.mock_hsm.keys.create(alias: :silver)
    signer.add_key(silver_key, chain.mock_hsm.signer_conn)

    # Account creation

    alice = chain.accounts.create(alias: :alice, root_xpubs: [alice_key.xpub], quorum: 1)
    bob = chain.accounts.create(alias: :bob, root_xpubs: [bob_key.xpub], quorum: 1)

    expect {
      # Request is missing key fields
      chain.accounts.create(alias: :david)
    }.to raise_error(Chain::APIError)

    # Batch account creation

    account_batch = chain.accounts.create_batch([
      {alias: :carol, root_xpubs: [chain.mock_hsm.keys.create.xpub], quorum: 1}, # success
      {alias: :david}, # error
      {alias: :eve, root_xpubs: [chain.mock_hsm.keys.create.xpub], quorum: 1}, #success
    ])

    expect(account_batch.errors.keys).to eq([1])
    expect(account_batch.successes.keys).to eq([0, 2])

    # Asset creation

    chain.assets.create(alias: :gold, root_xpubs: [gold_key.xpub], quorum: 1)
    chain.assets.create(alias: :silver, root_xpubs: [silver_key.xpub], quorum: 1)

    expect {
      # Request is missing key fields
      chain.assets.create(alias: :unobtanium)
    }.to raise_error(Chain::APIError)

    # Batch asset creation

    asset_batch = chain.assets.create_batch([
      {alias: :bronze, root_xpubs: [chain.mock_hsm.keys.create.xpub], quorum: 1}, # success
      {alias: :unobtanium}, # error
      {alias: :copper, root_xpubs: [chain.mock_hsm.keys.create.xpub], quorum: 1}, #success
    ])

    expect(asset_batch.errors.keys).to eq([1])
    expect(asset_batch.successes.keys).to eq([0, 2])

    # Basic issuance

    tx = chain.transactions.build do |b|
      b.issue asset_alias: :gold, amount: 100
      b.issue asset_alias: :silver, amount: 200
      b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 100
      b.control_with_account account_alias: :bob, asset_alias: :silver, amount: 200
    end

    chain.transactions.submit(signer.sign(tx))

    expect(
      balance_by_asset_alias(chain.balances.query(filter: "account_alias='alice'"))
    ).to eq('gold' => 100)

    expect(
      balance_by_asset_alias(chain.balances.query(filter: "account_alias='bob'"))
    ).to eq('silver' => 200)

    # Bad singleton build call

    expect {
      chain.transactions.build do |b|
        # Non-existent asset
        b.issue asset_alias: :unobtanium, amount: 100
      end
    }.to raise_error(Chain::APIError)

    # Bad singleton submit call

    unbalanced = signer.sign(chain.transactions.build { |b|
      b.issue asset_alias: :gold, amount: 1
      b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 100
    })

    expect { chain.transactions.submit(unbalanced) }.to raise_error(Chain::APIError)

    # Atomic swap

    swap_proposal = chain.transactions.build do |b|
      b.spend_from_account account_alias: :alice, asset_alias: :gold, amount: 10
      b.control_with_account account_alias: :alice, asset_alias: :silver, amount: 20
    end

    swap_proposal = signer.sign(swap_proposal.allow_additional_actions)

    swap_tx = chain.transactions.build do |b|
      b.base_transaction swap_proposal
      b.spend_from_account account_alias: :bob, asset_alias: :silver, amount: 20
      b.control_with_account account_alias: :bob, asset_alias: :gold, amount: 10
    end

    chain.transactions.submit(signer.sign(swap_tx))

    expect(
      balance_by_asset_alias(chain.balances.query(filter: "account_alias='alice'"))
    ).to eq('gold' => 90, 'silver' => 20)

    expect(
      balance_by_asset_alias(chain.balances.query(filter: "account_alias='bob'"))
    ).to eq('gold' => 10, 'silver' => 180)

    # Batch transactions

    builders = []

    # Should succeed
    builders << Chain::Transaction::Builder.new do |b|
      b.issue(asset_alias: :gold, amount: 100)
      b.control_with_account(account_alias: :alice, asset_alias: :gold, amount: 100)
    end

    # Should fail at the build step
    builders << Chain::Transaction::Builder.new do |b|
      b.issue(asset_alias: :foobar)
    end

    # Should fail at the submit step
    builders << Chain::Transaction::Builder.new do |b|
      b.issue(asset_alias: :gold, amount: 50)
      b.control_with_account(account_alias: :alice, asset_alias: :gold, amount: 100)
    end

    # Should succeed
    builders << Chain::Transaction::Builder.new do |b|
      b.issue(asset_alias: :silver, amount: 100)
      b.control_with_account(account_alias: :bob, asset_alias: :silver, amount: 100)
    end

    build_batch = chain.transactions.build_batch(builders)
    expect(build_batch.errors.keys).to eq([1])
    expect(build_batch.successes.keys).to eq([0, 2, 3])

    sign_batch = signer.sign_batch(build_batch.successes.values)
    expect(sign_batch.errors.keys).to eq([])
    expect(sign_batch.successes.keys).to eq([0, 1, 2])

    submit_batch = chain.transactions.submit_batch(sign_batch.successes.values)
    expect(submit_batch.errors.keys).to eq([1])
    expect(submit_batch.successes.keys).to eq([0, 2])

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

  example 'account tag updates' do
    chain = Chain::Client.new
    k = chain.mock_hsm.keys.create
    acc1 = chain.accounts.create(root_xpubs: [k.xpub], quorum: 1, tags: {x: 'one'})
    acc2 = chain.accounts.create(root_xpubs: [k.xpub], quorum: 1, tags: {y: 'one'})

    # Account tag update
    chain.accounts.update_tags(id: acc1.id, tags: {x: 'two'})
    expect(
      chain.accounts.query(filter: "id='#{acc1.id}'").first.tags
    ).to eq('x' => 'two')

    expect {
      chain.accounts.update_tags(tags: {x: 'three'}) # ID intentionally omitted
    }.to raise_error(Chain::APIError)

    # Batch account tag update

    chain.accounts.update_tags_batch([
      {id: acc1.id, tags: {x: 'four'}},
      {id: acc2.id, tags: {y: 'four'}},
    ])
    expect(
      chain.accounts.query(
        filter: "id=$1 OR id=$2",
        filter_params: [acc1.id, acc2.id]
      ).all.map(&:tags).reverse
    ).to eq([
      {'x' => 'four'},
      {'y' => 'four'},
    ])

    expect(
      chain.accounts.update_tags_batch([
        {id: acc1.id, alias: :redundant_alias, tags: {x: 'five'}},
        {tags: {y: 'five'}}, # ID intentionally omitted
      ]).errors.size
    ).to eq(2)
  end

  example 'asset tag updates' do
    chain = Chain::Client.new
    k = chain.mock_hsm.keys.create
    asset1 = chain.assets.create(root_xpubs: [k.xpub], quorum: 1, tags: {x: 'one'})
    asset2 = chain.assets.create(root_xpubs: [k.xpub], quorum: 1, tags: {y: 'one'})

    # Account tag update
    chain.assets.update_tags(id: asset1.id, tags: {x: 'two'})
    expect(
      chain.assets.query(filter: "id='#{asset1.id}'").first.tags
    ).to eq('x' => 'two')

    expect {
      chain.assets.update_tags(tags: {x: 'three'}) # ID intentionally omitted
    }.to raise_error(Chain::APIError)

    # Batch asset tag update

    chain.assets.update_tags_batch([
      {id: asset1.id, tags: {x: 'four'}},
      {id: asset2.id, tags: {y: 'four'}},
    ])
    expect(
      chain.assets.query(
        filter: "id=$1 OR id=$2",
        filter_params: [asset1.id, asset2.id]
      ).all.map(&:tags).reverse
    ).to eq([
      {'x' => 'four'},
      {'y' => 'four'},
    ])

    expect(
      chain.assets.update_tags_batch([
        {id: asset1.id, alias: :redundant_alias, tags: {x: 'five'}},
        {tags: {y: 'five'}}, # ID intentionally omitted
      ]).errors.size
    ).to eq(2)
  end

  example 'authorization grants' do
    chain = Chain::Client.new

    # setup: delete all existing guards

    chain.authorization_grants.list_all.each do |g|
      chain.authorization_grants.delete g unless g.protected
    end

    # Access token grant

    t = chain.access_tokens.create(id: SecureRandom.hex(8))

    g = chain.authorization_grants.create(
      guard_type: 'access_token',
      guard_data: {'id' => t.id},
      policy: 'client-readwrite'
    )

    expect(g.guard_type).to eq('access_token')
    expect(g.guard_data).to eq('id' => t.id)
    expect(g.policy).to eq('client-readwrite')

    # Listing

    g = chain.authorization_grants.list_all.find { |g| g.guard_data['id'] == t.id && !g.protected }
    expect(g.guard_type).to eq('access_token')
    expect(g.policy).to eq('client-readwrite')

    # X509 grant

    chain.authorization_grants.create(
      guard_type: 'x509',
      guard_data: {
        'subject' => {
          'CN' => 'test-cn',
          'OU' => 'test-ou',
        }
      },
      policy: 'crosscore'
    )

    guards = chain.authorization_grants.list_all
    g = guards.find { |g| g.guard_type == 'x509' && !g.protected }

    expect(g.guard_data).to eq('subject' => {
      'CN' => 'test-cn',
      'OU' => ['test-ou'], # sanitizer properly array-ifies attributes
    })
    expect(g.policy).to eq('crosscore')

    # Deletion

    chain.authorization_grants.delete(
      guard_type: 'access_token',
      guard_data: {'id' => t.id},
      policy: 'client-readwrite'
    )

    guards = chain.authorization_grants.list_all.select { |g| g.guard_type == 'access_token' }
    expect(guards).to be_empty
  end

end
