require 'chain'

# snippet create-client
chain = Chain::Client.new
# endsnippet

# snippet create-key
key = chain.mock_hsm.keys.create
# endsnippet

# snippet signer-add-key
signer.add_key(key, chain.mock_hsm.signer_conn)
# endsnippet

# snippet create-asset
chain.assets.create(
  alias: 'gold',
  root_xpubs: [key.xpub],
  quorum: 1,
)
# endsnippet

# snippet create-account-alice
chain.accounts.create(
  alias: 'alice',
  root_xpubs: [key.xpub],
  quorum: 1,
)
# endsnippet

# snippet create-account-bob
chain.accounts.create(
  alias: 'bob',
  root_xpubs: [key.xpub],
  quorum: 1,
)
# endsnippet

# snippet issue
issuance = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)

chain.transactions.submit(signer.sign(issuance))
# endsnippet

# snippet spend
spending = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(10))
  .addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(10)
  ).build(client)

chain.transactions.submit(signer.sign(spending))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(5)
  ).addAction(new Transaction.Action.Retire()
    .setAssetAlias('gold')
    .setAmount(5)
  ).build(client)

chain.transactions.submit(signer.sign(retirement))
# endsnippet
