require 'chain'

chain = Chain::Client.new

# snippet create-key
key = chain.mock_hsm.keys.create
# endsnippet

# snippet signer-add-key
signer.add_key(key, chain.mock_hsm.signer_conn)
# endsnippet

chain.assets.create()
  .setAlias('gold')
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .create(client)

chain.accounts.create()
  .setAlias('alice')
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .create(client)

unsigned = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)

# snippet sign-transaction
signed = signer.sign(unsigned)
# endsnippet
