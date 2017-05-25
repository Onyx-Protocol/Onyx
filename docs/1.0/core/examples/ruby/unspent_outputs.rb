require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create(alias: 'gold', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'alice', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'bob', root_xpubs: [key.xpub], quorum: 1)

issuance_tx = chain.transactions.submit(signer.sign(chain.transactions.build { |b|
  b.issue asset_alias: 'gold', amount: 200
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
}))

# snippet alice-unspent-outputs
alice_unspent_outputs = chain.unspent_outputs.query(
  filter: 'account_alias=$1',
  filter_params: ['alice'],
).each do |utxo|
  puts "Unspent output in alice account: #{utxo.transaction_id}:#{utxo.position}"
end
# endsnippet

# snippet gold-unspent-outputs
goldUnspentOutputs = chain.unspent_outputs.query(
  filter: 'asset_alias=$1',
  filter_params: ['gold'],
).each do |utxo|
  puts "Unspent output containing gold: #{utxo.transaction_id}:#{utxo.position}"
end
# endsnippet

prev_transaction_id = issuance_tx.id

# snippet build-transaction-all
spend_output = chain.transactions.build do |b|
  b.spend_account_unspent_output transaction_id: prev_transaction_id, position: 0
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 100
end
# endsnippet

chain.transactions.submit(signer.sign(spend_output))

# snippet build-transaction-partial
spend_output_with_change = chain.transactions.build do |b|
  b.spend_account_unspent_output transaction_id: prev_transaction_id, position: 1
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 40
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 60
end
# endsnippet

chain.transactions.submit(signer.sign(spend_output_with_change))
