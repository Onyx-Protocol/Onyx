require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

asset_key = chain.mock_hsm.keys.create
signer.add_key(asset_key, chain.mock_hsm.signer_conn)

account_key = chain.mock_hsm.keys.create
signer.add_key(account_key, chain.mock_hsm.signer_conn)

chain.accounts.create(
  alias: 'acme_treasury',
  root_xpubs: [account_key.xpub],
  quorum: 1,
)

# snippet create-asset-acme-common
chain.assets.create(
  alias: 'acme_common',
  root_xpubs: [asset_key.xpub],
  quorum: 1,
  tags: {
    internal_rating: '1',
  },
  definition: {
    issuer: 'Acme Inc.',
    type: 'security',
    subtype: 'private',
    class: 'common',
  },
)
# endsnippet

# snippet create-asset-acme-preferred
chain.assets.create(
  alias: 'acme_preferred',
  root_xpubs: [asset_key.xpub],
  quorum: 1,
  tags: {
    internal_rating: '2',
  },
  definition: {
    issuer: 'Acme Inc.',
    type: 'security',
    subtype: 'private',
    class: 'preferred',
  },
)
# endsnippet

# snippet list-local-assets
chain.assets.query(
  filter: 'is_local=$1',
  filter_params: ['yes'],
).each do |a|
  puts "Local asset: #{a.alias}"
end
# endsnippet

# snippet list-private-preferred-securities
chain.assets.query(
  filter: 'definition.type=$1 AND definition.subtype=$2 AND definition.class=$3',
  filter_params: ['security', 'private', 'preferred'],
).each do |a|
  puts "Private preferred security: #{a.alias}"
end
# endsnippet

# snippet build-issue
issuance_tx = chain.transactions.build do |b|
  b.issue asset_alias: 'acme_common', amount: 1000
  b.control_with_account account_alias: 'acme_treasury', asset_alias: 'acme_common', amount: 1000
end
# endsnippet

# snippet sign-issue
signed_issuance_tx = signer.sign(issuance_tx)
# endsnippet

# snippet submit-issue
chain.transactions.submit(signed_issuance_tx)
# endsnippet

external_program = chain.accounts.create_control_program(
  alias: 'acme_treasury'
).control_program

# snippet external-issue
external_issuance = chain.transactions.build do |b|
  b.issue asset_alias: 'acme_preferred', amount: 2000
  b.control_with_program control_program: external_program, asset_alias: 'acme_preferred', amount: 2000
end

chain.transactions.submit(signer.sign(external_issuance))
# endsnippet

# snippet build-retire
retirement_tx = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'acme_treasury', asset_alias: 'acme_common', amount: 50
  b.retire asset_alias: 'acme_common', amount: 50
end
# endsnippet

# snippet sign-retire
signed_retirement_tx = signer.sign(retirement_tx)
# endsnippet

# snippet submit-retire
chain.transactions.submit(signed_retirement_tx)
# endsnippet

# snippet list-issuances
chain.transactions.query(
  filter: 'inputs(type=$1 AND asset_alias=$2)',
  filter_params: ['issue', 'acme_common'],
).each do |t|
  puts "Acme Common issued in tx #{t.id}"
end
# endsnippet

# snippet list-transfers
chain.transactions.query(
  filter: 'inputs(type=$1 AND asset_alias=$2)',
  filter_params: ['spend', 'acme_common'],
).each do |t|
  puts "Acme Common transferred in tx #{t.id}"
end
# endsnippet

# snippet list-retirements
chain.transactions.query(
  filter: 'outputs(type=$1 AND asset_alias=$2)',
  filter_params: ['retire', 'acme_common'],
).each do |t|
  puts "Acme Common retired in tx #{t.id}"
end
# endsnippet

# snippet list-acme-common-balance
chain.balances.query(
  filter: 'asset_alias=$1',
  filter_params: ['acme_common'],
).each do |b|
  puts "Total circulation of Acme Common: #{b.amount}"
end
# endsnippet

# snippet list-acme-balance
chain.balances.query(
  filter: 'asset_definition.issuer=$1',
  filter_params: ['Acme Inc.'],
).each do |b|
  puts "Total circulation of Acme stock #{b.sum_by['asset_alias']}: #{b.amount}"
end
# endsnippet

# snippet list-acme-common-unspents
chain.unspent_outputs.query(
  filter: 'asset_alias=$1',
  filter_params: ['acme_common'],
).each do |u|
  puts "Acme Common held in output #{u.transaction_id}:#{u.position}"
end
# endsnippet
