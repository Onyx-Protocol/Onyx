require 'chain'

# snippet create-client
chain = Chain::Client.new
# endsnippet

# snippet create-key
key = chain.mock_hsm.keys.create
# endsnippet

# snippet signer-add-key
signer = Chain::HSMSigner.new
signer.add_key(key, chain.mock_hsm.signer_conn)
# endsnippet

# snippet create-asset
chain.assets.create(
  alias: :gold,
  root_xpubs: [key.xpub],
  quorum: 1
)
# endsnippet

# snippet create-account-alice
chain.accounts.create(
  alias: :alice,
  root_xpubs: [key.xpub],
  quorum: 1
)
# endsnippet

# snippet create-account-bob
chain.accounts.create(
  alias: :bob,
  root_xpubs: [key.xpub],
  quorum: 1
)
# endsnippet

# snippet issue
issuance = chain.transactions.build do |b|
  b.issue asset_alias: :gold, amount: 100
  b.control_with_account account_alias: :alice, asset_alias: :gold, amount: 100
end

chain.transactions.submit(signer.sign(issuance))
# endsnippet

# snippet spend
spending = chain.transactions.build do |b|
  b.spend_from_account account_alias: :alice, asset_alias: :gold, amount: 10
  b.control_with_account account_alias: :bob, asset_alias: :gold, amount: 10
end

chain.transactions.submit(signer.sign(spending))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account account_alias: :bob, asset_alias: :gold, amount: 5
  b.retire asset_alias: :gold, amount: 5
end

chain.transactions.submit(signer.sign(retirement))
# endsnippet
