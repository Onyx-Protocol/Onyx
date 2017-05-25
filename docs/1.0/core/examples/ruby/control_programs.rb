require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create(
  alias: 'gold',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'alice',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'bob',
  root_xpubs: [key.xpub],
  quorum: 1,
)

tx = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 100
end

chain.transactions.submit(signer.sign(tx))

# snippet create-control-program
alice_program = chain.accounts.create_control_program(
  alias: 'alice'
).control_program
# endsnippet

# snippet build-transaction
payment_to_program = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'bob', asset_alias: 'gold', amount: 10
  b.control_with_program control_program: alice_program, asset_alias: 'gold', amount: 10
end

chain.transactions.submit(signer.sign(payment_to_program))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.retire asset_alias: 'gold', amount: 10
end

chain.transactions.submit(signer.sign(retirement))
# endsnippet
