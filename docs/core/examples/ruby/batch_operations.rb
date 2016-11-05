require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

# snippet asset-builders
assets_to_build = [{
  alias: 'gold',
  root_xpubs: [key.xpub],
  quorum: 1,
}, {
  alias: 'silver',
  root_xpubs: [key.xpub],
  quorum: 1,
}, {
  alias: 'bronze',
  root_xpubs: [key.xpub],
  quorum: 0,
}]
# endsnippet

# snippet asset-create-batch
asset_batch = chain.assets.create_batch(assets_to_build)
# endsnippet

# snippet asset-create-handle-errors
asset_batch.errors.each do |index, err|
  puts "asset #{index} error: #{err}"
end

asset_batch.successes.each do |index, asset|
  puts "asset #{index} created, ID: #{asset.id}"
end
# endsnippet

# snippet nondeterministic-errors
assets_to_build = [{
  alias: 'platinum',
  root_xpubs: [key.xpub],
  quorum: 1,
}, {
  alias: 'platinum',
  root_xpubs: [key.xpub],
  quorum: 1,
}, {
  alias: 'platinum',
  root_xpubs: [key.xpub],
  quorum: 1,
}]
# endsnippet

asset_batch = chain.assets.create_batch(assets_to_build)

asset_batch.errors.each do |index, err|
  puts "asset #{index} error: #{err}"
end

asset_batch.successes.each do |index, asset|
  puts "asset #{index} created, ID: #{asset.id}"
end

# Some setup for the next examples.

chain.accounts.create(alias: 'alice', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'bob', root_xpubs: [key.xpub], quorum: 1)

# snippet batch-build-builders
transactions_to_build = []

transactions_to_build << Chain::Transaction::Builder.new do |b|
  b.issue asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
end

transactions_to_build << Chain::Transaction::Builder.new do |b|
  b.issue asset_alias: 'not-a-real-asset', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'not-a-real-asset', amount: 100
end

transactions_to_build << Chain::Transaction::Builder.new do |b|
  b.issue asset_alias: 'silver', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'silver', amount: 100
end
# endsnippet

# snippet batch-build-handle-errors
build_batch = chain.transactions.build_batch(transactions_to_build)

build_batch.errors.each do |index, err|
  puts "Error building transaction #{index}: #{err}"
end
# endsnippet

# snippet batch-sign
transactions_to_sign = build_batch.successes.values
sign_batch = signer.sign_batch(transactions_to_sign)

sign_batch.errors.each do |index, err|
  puts "Error signing transaction #{index}: #{err}"
end
# endsnippet

# snippet batch-submit
transactions_to_submit = sign_batch.successes.values
submit_batch = chain.transactions.submit_batch(transactions_to_submit)

submit_batch.errors.each do |index, err|
  puts "Error submitting transaction #{index}: #{err}"
end

submit_batch.successes.each do |index, submission|
  puts "Transaction #{index} submitted, ID: #{submission.id}"
end
# endsnippet
