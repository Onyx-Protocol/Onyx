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

issuance_tx = chain.transactions.submit(
  client,
  signer.sign(
    chain.transactions.build do |b|
      b.issue asset_alias: 'gold', amount: 200
      b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
      b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
    end
  )
)

# snippet alice-unspent-outputs
aliceUnspentOutputs = chain.unspent_outputs.query()
  filter: 'account_alias=$1',
  filter_params: ['alice'],
  .execute(client)

while (aliceUnspentOutputs.hasNext()) {
  UnspentOutput utxo = aliceUnspentOutputs.next()
  puts('Unspent output in alice account: ' + utxo.transaction_id + ':' + utxo.position)
}
# endsnippet

# snippet gold-unspent-outputs
goldUnspentOutputs = chain.unspent_outputs.query()
  filter: 'asset_alias=$1',
  filter_params: ['gold'],
  .execute(client)

while (goldUnspentOutputs.hasNext()) {
  UnspentOutput utxo = goldUnspentOutputs.next()
  puts('Unspent output containing gold: ' + utxo.transaction_id + ':' + utxo.position)
}
# endsnippet

String prevTransactionId = issuance_tx.id

# snippet build-transaction-all
spendOutput = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId(prevTransactionId)
    .setPosition(0)
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 100,
  ).build(client)
# endsnippet

chain.transactions.submit(signer.sign(spendOutput))

# snippet build-transaction-partial
spendOutputWithChange = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendAccountUnspentOutput()
    .setTransactionId(prevTransactionId)
    .setPosition(1)
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 40,
  b.control_with_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 60,
  ).build(client)
# endsnippet

chain.transactions.submit(signer.sign(spendOutputWithChange))
