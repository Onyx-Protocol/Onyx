require 'chain'

chain = Chain::Client.new

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

issuanceTx = chain.transactions.submit(
  client,
  signer.sign(
    chain.transactions.build do |b|
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias('gold')
        .setAmount(200)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias('alice')
        .setAssetAlias('gold')
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias('alice')
        .setAssetAlias('gold')
        .setAmount(100)
      ).build(client)
  )
)

# snippet alice-unspent-outputs
UnspentOutput.Items aliceUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter('account_alias=$1')
  .addFilterParameter('alice')
  .execute(client)

while (aliceUnspentOutputs.hasNext()) {
  UnspentOutput utxo = aliceUnspentOutputs.next()
  puts('Unspent output in alice account: ' + utxo.transactionId + ':' + utxo.position)
}
# endsnippet

# snippet gold-unspent-outputs
UnspentOutput.Items goldUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter('asset_alias=$1')
  .addFilterParameter('gold')
  .execute(client)

while (goldUnspentOutputs.hasNext()) {
  UnspentOutput utxo = goldUnspentOutputs.next()
  puts('Unspent output containing gold: ' + utxo.transactionId + ':' + utxo.position)
}
# endsnippet

String prevTransactionId = issuanceTx.id

# snippet build-transaction-all
spendOutput = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId(prevTransactionId)
    .setPosition(0)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)
# endsnippet

chain.transactions.submit(signer.sign(spendOutput))

# snippet build-transaction-partial
spendOutputWithChange = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId(prevTransactionId)
    .setPosition(1)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(40)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(60)
  ).build(client)
# endsnippet

chain.transactions.submit(signer.sign(spendOutputWithChange))
