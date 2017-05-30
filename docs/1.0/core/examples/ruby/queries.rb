require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create(alias: 'gold', root_xpubs: [key.xpub], quorum: 1)
chain.assets.create(alias: 'silver', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'alice', tags: {type: 'checking'}, root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'bob', root_xpubs: [key.xpub], quorum: 1)

chain.transactions.submit(signer.sign(chain.transactions.build { |b|
  b.issue asset_alias: 'gold', amount: 1000
  b.issue asset_alias: 'silver', amount: 1000
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 1000
  b.control_with_account account_alias: 'bob', asset_alias: 'silver', amount: 1000
}))

chain.transactions.submit(signer.sign(chain.transactions.build { |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.spend_from_account account_alias: 'bob', asset_alias: 'silver', amount: 10
  b.control_with_account account_alias: 'alice', asset_alias: 'silver', amount: 10
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 10
}))

chain.assets.create(alias: 'bank1_usd_iou', root_xpubs: [key.xpub], quorum: 1, definition: {currency: 'USD'})
chain.assets.create(alias: 'bank1_euro_iou', root_xpubs: [key.xpub], quorum: 1, definition: {currency: 'Euro'})
chain.assets.create(alias: 'bank2_usd_iou', root_xpubs: [key.xpub], quorum: 1, definition: {currency: 'USD'})
chain.accounts.create(alias: 'bank1', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'bank2', root_xpubs: [key.xpub], quorum: 1)

chain.transactions.submit(signer.sign(chain.transactions.build { |b|
  b.issue asset_alias: 'bank1_usd_iou', amount: 2000000
  b.issue asset_alias: 'bank2_usd_iou', amount: 2000000
  b.issue asset_alias: 'bank1_euro_iou', amount: 2000000
  b.control_with_account account_alias: 'bank1', asset_alias: 'bank1_usd_iou', amount: 1000000
  b.control_with_account account_alias: 'bank1', asset_alias: 'bank1_euro_iou', amount: 1000000
  b.control_with_account account_alias: 'bank1', asset_alias: 'bank2_usd_iou', amount: 1000000
  b.control_with_account account_alias: 'bank2', asset_alias: 'bank1_usd_iou', amount: 1000000
  b.control_with_account account_alias: 'bank2', asset_alias: 'bank1_euro_iou', amount: 1000000
  b.control_with_account account_alias: 'bank2', asset_alias: 'bank2_usd_iou', amount: 1000000
}))

# snippet list-alice-transactions
alice_txs = chain.transactions.query(
  filter: 'inputs(account_alias=$1) OR outputs(account_alias=$1)',
  filter_params: ['alice'],
).each do |tx|
  puts "Alice's transaction: #{tx.id}"

  tx.inputs.each do |input|
    puts "-#{input.amount} #{input.asset_alias}"
  end

  tx.outputs.each do |output|
    puts "+#{output.amount} #{output.asset_alias}"
  end
end
# endsnippet

# snippet list-local-transactions
chain.transactions.query(
  filter: 'is_local=$1',
  filter_params: ['yes'],
).each do |tx|
  puts "Local transaction #{tx.id}"
end
# endsnippet

# snippet list-local-assets
chain.assets.query(
  filter: 'is_local=$1',
  filter_params: ['yes'],
).each do |asset|
  puts "Local asset #{asset.id} (#{asset.alias})"
end
# endsnippet

# snippet list-usd-assets
chain.assets.query(
  filter: 'definition.currency=$1',
  filter_params: ['USD'],
).each do |asset|
  puts "USD asset #{asset.id} (#{asset.alias})"
end
# endsnippet

# snippet list-checking-accounts
chain.accounts.query(
  filter: 'tags.type=$1',
  filter_params: ['checking'],
).each do |account|
  puts "Checking account #{account.id} (#{account.alias})"
end
# endsnippet

# snippet list-alice-unspents
chain.unspent_outputs.query(
  filter: 'account_alias=$1',
  filter_params: ['alice'],
).each do |utxo|
  puts "Alice's unspent output: #{utxo.amount} #{utxo.asset_alias}"
end
# endsnippet

# snippet account-balance
chain.balances.query(
  filter: 'account_alias=$1',
  filter_params: ['bank1'],
).each do |b|
  puts "Bank 1 balance of #{b.sum_by['asset_alias']}: #{b.amount}"
end
# endsnippet

# snippet usd-iou-circulation
circulation = chain.balances.query(
  filter: 'asset_alias=$1',
  filter_params: ['bank1_usd_iou'],
).first

puts("Total circulation of Bank 1 USD IOU:  #{circulation.amount}")
# endsnippet

# snippet account-balance-sum-by-currency
chain.balances.query(
  filter: 'account_alias=$1',
  filter_params: ['bank1'],
  sum_by: ['asset_definition.currency']
).each do |b|
  denom = b.sum_by['asset_definition.currency']
  puts "Bank 1 balance of #{denom}-denominated currencies: #{b.amount}"
end
# endsnippet
