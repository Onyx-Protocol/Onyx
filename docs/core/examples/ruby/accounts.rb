require 'chain'

chain = Chain::Client.new

asset_key = chain.mock_hsm.keys.create
signer.add_key(asset_key, chain.mock_hsm.signer_conn)

alice_key = chain.mock_hsm.keys.create
signer.add_key(alice_key, chain.mock_hsm.signer_conn)

bob_key = chain.mock_hsm.keys.create
signer.add_key(bob_key, chain.mock_hsm.signer_conn)

chain.assets.create()
  .setAlias('gold')
  .addRootXpub(asset_key.xpub)
  .setQuorum(1)
  .create(client)

chain.assets.create()
  .setAlias('silver')
  .addRootXpub(asset_key.xpub)
  .setQuorum(1)
  .create(client)

# snippet create-account-alice
chain.accounts.create()
  .setAlias('alice')
  .addRootXpub(alice_key.xpub)
  .setQuorum(1)
  .addTag('type', 'checking')
  .addTag('first_name', 'Alice')
  .addTag('last_name', 'Jones')
  .addTag('user_id', '12345')
  .create(client)
# endsnippet

# snippet create-account-bob
chain.accounts.create()
  .setAlias('bob')
  .addRootXpub(bob_key.xpub)
  .setQuorum(1)
  .addTag('type', 'savings')
  .addTag('first_name', 'Bob')
  .addTag('last_name', 'Smith')
  .addTag('user_id', '67890')
  .create(client)
# endsnippet

# snippet list-accounts-by-tag
accounts = chain.accounts.query
  .setFilter('tags.type=$1')
  .addFilterParameter('savings')
  .execute(client)

while (accounts.hasNext()) {
  Account a = accounts.next()
  puts('Account ID ' + a.id + ', alias ' + a.alias)
}
# endsnippet

fundAliceTransaction = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)

chain.transactions.submit(signer.sign(fundAliceTransaction))

fundBobTransaction = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('silver')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('silver')
    .setAmount(100)
  ).build(client)

chain.transactions.submit(signer.sign(fundBobTransaction))

# snippet build-transfer
spendingTransaction = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(10)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(10)
  ).build(client)
# endsnippet

# snippet sign-transfer
signedSpendingTransaction = signer.sign(spendingTransaction)
# endsnippet

# snippet submit-transfer
chain.transactions.submit(signedSpendingTransaction)
# endsnippet

# snippet create-control-program
ControlProgram bobProgram = chain.accounts.create_control_program()
  .controlWithAccountByAlias('bob')
  .create(client)
# endsnippet

# snippet transfer-to-control-program
spendingTransaction2 = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(10)
  ).addAction(new Transaction.Action.ControlWithProgram()
    .setControlProgram(bobProgram)
    .setAssetAlias('gold')
    .setAmount(10)
  ).build(client)

chain.transactions.submit(signer.sign(spendingTransaction2))
# endsnippet

# snippet list-account-txs
transactions = chain.transactions.query
  .setFilter('inputs(account_alias=$1) AND outputs(account_alias=$1)')
  .addFilterParameter('alice')
  .execute(client)

while (transactions.hasNext()) {
  t = transactions.next()
  puts('' + t.id + ' at ' + t.timestamp)
}
# endsnippet

# snippet list-account-balances
balances = chain.balances.query
  .setFilter('account_alias=$1')
  .addFilterParameter('alice')
  .execute(client)

while (balances.hasNext()) {
  b = balances.next()
  puts(
    'Alice\'s balance of ' + b.sumBy.get('asset_alias') +
    ': ' + b.amount
  )
}
# endsnippet

# snippet list-account-unspent-outputs
UnspentOutput.Items unspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter('account_alias=$1 AND asset_alias=$2')
  .addFilterParameter('alice')
  .addFilterParameter('gold')
  .execute(client)

while (unspentOutputs.hasNext()) {
  UnspentOutput u = unspentOutputs.next()
  puts('' + u.transactionId + ' position ' + u.position)
}
# endsnippet
