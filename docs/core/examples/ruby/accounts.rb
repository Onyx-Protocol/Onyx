require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

asset_key = chain.mock_hsm.keys.create
signer.add_key(asset_key, chain.mock_hsm.signer_conn)

alice_key = chain.mock_hsm.keys.create
signer.add_key(alice_key, chain.mock_hsm.signer_conn)

bob_key = chain.mock_hsm.keys.create
signer.add_key(bob_key, chain.mock_hsm.signer_conn)

chain.assets.create(
  alias: 'gold',
  root_xpubs: [asset_key.xpub],
  quorum: 1,
)

chain.assets.create(
  alias: 'silver',
  root_xpubs: [asset_key.xpub],
  quorum: 1,
)

# snippet create-account-alice
chain.accounts.create(
  alias: 'alice',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
  tags: {
    type: 'checking',
    first_name: 'Alice',
    last_name: 'Jones',
    user_id: '12345',
  }
)
# endsnippet

# snippet create-account-bob
chain.accounts.create(
  alias: 'bob',
  root_xpubs: [bob_key.xpub],
  quorum: 1,
  tags: {
    type: 'savings',
    first_name: 'Bob',
    last_name: 'Smith',
    user_id: '67890',
  }
)
# endsnippet

# snippet list-accounts-by-tag
chain.accounts.query(
  filter: 'tags.type=$1',
  filter_params: ['savings'],
).each do |a|
  puts "Account ID #{a.id} alias #{a.alias}"
end
# endsnippet

fund_alice_tx = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
end

chain.transactions.submit(signer.sign(fund_alice_tx))

fund_bob_tx = chain.transactions.build do |b|
  b.issue asset_alias: 'silver', amount: 100
  b.control_with_account account_alias: 'bob', asset_alias: 'silver', amount: 100
end

chain.transactions.submit(signer.sign(fund_bob_tx))

# snippet build-transfer
spending_tx = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 10
end
# endsnippet

# snippet sign-transfer
signed_spending_tx = signer.sign(spending_tx)
# endsnippet

# snippet submit-transfer
chain.transactions.submit(signed_spending_tx)
# endsnippet

# snippet create-control-program
bob_program = chain.accounts.create_control_program(
  alias: 'bob'
).control_program
# endsnippet

# snippet transfer-to-control-program
spending_tx2 = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.control_with_program control_program: bob_program, asset_alias: 'gold', amount: 10
end

chain.transactions.submit(signer.sign(spending_tx2))
# endsnippet

# snippet list-account-txs
transactions = chain.transactions.query(
  filter: 'inputs(account_alias=$1) AND outputs(account_alias=$1)',
  filter_params: ['alice'],
).each do |t|
  puts "#{t.id} at #{t.timestamp}"
end
# endsnippet

# snippet list-account-balances
balances = chain.balances.query(
  filter: 'account_alias=$1',
  filter_params: ['alice'],
).each do |b|
  puts "Alice's balance of #{b.sum_by['asset_alias']}: #{b.amount}"
end
# endsnippet

# snippet list-account-unspent-outputs
unspentOutputs = chain.unspent_outputs.query(
  filter: 'account_alias=$1 AND asset_alias=$2',
  filter_params: ['alice', 'gold'],
).each do |u|
  puts "#{u.transaction_id} position #{u.position}"
end
# endsnippet
