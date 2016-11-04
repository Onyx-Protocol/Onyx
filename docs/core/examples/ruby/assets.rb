require 'chain'

chain = Chain::Client.new

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
  .addDefinitionField('issuer', 'Acme Inc.')
  .addDefinitionField('type', 'security')
  .addDefinitionField('subtype', 'private')
  .addDefinitionField('class', 'common')
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
  .addDefinitionField('issuer', 'Acme Inc.')
  .addDefinitionField('type', 'security')
  .addDefinitionField('subtype', 'private')
  .addDefinitionField('class', 'perferred')
)
# endsnippet

# snippet list-local-assets
localAssets = chain.assets.query
  filter: 'is_local=$1',
  filter_params: ['yes'],
  .execute(client)

while (localAssets.hasNext()) {
  Asset asset = localAssets.next()
  puts('Local asset: ' + asset.alias)
}
# endsnippet

# snippet list-private-preferred-securities
common = chain.assets.query
  filter: 'definition.type=$1 AND definition.subtype=$2 AND definition.class=$3',
  filter_params: ['security'],
  filter_params: ['private'],
  filter_params: ['preferred'],
  .execute(client)
# endsnippet

# snippet build-issue
issuanceTransaction = chain.transactions.build do |b|
  b.issue
    asset_alias: 'acme_common',
    amount: 1000,
  b.control_with_account
    account_alias: 'acme_treasury',
    asset_alias: 'acme_common',
    amount: 1000,
  ).build(client)
# endsnippet

# snippet sign-issue
signedIssuanceTransaction = signer.sign(issuanceTransaction)
# endsnippet

# snippet submit-issue
chain.transactions.submit(signedIssuanceTransaction)
# endsnippet

externalProgram = chain.accounts.create_control_program()
  alias: 'acme_treasury'
)

# snippet external-issue
externalIssuance = chain.transactions.build do |b|
  b.issue
    asset_alias: 'acme_preferred',
    amount: 2000,
  )b.control_with_program
    control_program: externalProgram,
    asset_alias: 'acme_preferred',
    amount: 2000,
  ).build(client)

chain.transactions.submit(signer.sign(externalIssuance))
# endsnippet

# snippet build-retire
retirementTransaction = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'acme_treasury',
    asset_alias: 'acme_common',
    amount: 50,
  ).addAction(new Transaction.Action.Retire()
    asset_alias: 'acme_common',
    amount: 50,
  ).build(client)
# endsnippet

# snippet sign-retire
signedRetirementTransaction = signer.sign(retirementTransaction)
# endsnippet

# snippet submit-retire
chain.transactions.submit(signedRetirementTransaction)
# endsnippet

# snippet list-issuances
acmeCommonIssuances = chain.transactions.query
  filter: 'inputs(action=$1 AND asset_alias=$2,')
  filter_params: ['issue'],
  filter_params: ['acme_common'],
  .execute(client)

while (acmeCommonIssuances.hasNext()) {
  tx = acmeCommonIssuances.next()
  puts('Acme Common issued in tx ' + tx.id)
}
# endsnippet

# snippet list-transfers
acmeCommonTransfers = chain.transactions.query
  filter: 'inputs(action=$1 AND asset_alias=$2,')
  filter_params: ['spend'],
  filter_params: ['acme_common'],
  .execute(client)

while (acmeCommonTransfers.hasNext()) {
  tx = acmeCommonTransfers.next()
  puts('Acme Common transferred in tx ' + tx.id)
}
# endsnippet

# snippet list-retirements
acmeCommonRetirements = chain.transactions.query
  filter: 'outputs(action=$1 AND asset_alias=$2,')
  filter_params: ['retire'],
  filter_params: ['acme_common'],
  .execute(client)

while (acmeCommonRetirements.hasNext()) {
  tx = acmeCommonRetirements.next()
  puts('Acme Common retired in tx ' + tx.id)
}
# endsnippet

# snippet list-acme-common-balance
acmeCommonBalances = chain.balances.query
  filter: 'asset_alias=$1',
  filter_params: ['acme_common'],
  .execute(client)

acmeCommonBalance = acmeCommonBalances.next()
puts('Total circulation of Acme Common: ' + acmeCommonBalance.amount)
# endsnippet

# snippet list-acme-balance
acmeAnyBalances = chain.balances.query
  filter: 'asset_definition.issuer=$1',
  filter_params: ['Acme Inc.'],
  .execute(client)

while (acmeAnyBalances.hasNext()) {
  stockBalance = acmeAnyBalances.next()
  puts(
    'Total circulation of Acme stock ' + stockBalance.sumBy.get('asset_alias') +
    ': ' + stockBalance.amount
  )
}
# endsnippet

# snippet list-acme-common-unspents
acmeCommonUnspentOutputs = chain.unspent_outputs.query()
  filter: 'asset_alias=$1',
  filter_params: ['acme_common'],
  .execute(client)

while (acmeCommonUnspentOutputs.hasNext()) {
  UnspentOutput utxo = acmeCommonUnspentOutputs.next()
  puts('Acme Common held in output ' + utxo.transaction_id + ':' + utxo.position)
}
# endsnippet
