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

# snippet create-receiver
alice_receiver = chain.accounts.create_receiver(
  account_alias: 'alice'
)
alice_receiver_serialized = alice_receiver.to_json
# endsnippet

# snippet build-transaction
payment_to_receiver = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'bob', asset_alias: 'gold', amount: 10
  b.control_with_receiver(
    receiver: JSON.parse(alice_receiver_serialized),
    asset_alias: 'gold',
    amount: 10
  )
end

chain.transactions.submit(signer.sign(payment_to_receiver))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.retire asset_alias: 'gold', amount: 10
end

chain.transactions.submit(signer.sign(retirement))
# endsnippet
