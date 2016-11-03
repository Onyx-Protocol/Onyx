require 'chain'

chain = Chain::Client.new

asset_key = chain.mock_hsm.keys.create
signer.add_key(asset_key, chain.mock_hsm.signer_conn)

account_key = chain.mock_hsm.keys.create
signer.add_key(account_key, chain.mock_hsm.signer_conn)

chain.accounts.create()
  .setAlias('acme_treasury')
  .addRootXpub(account_key.xpub)
  .setQuorum(1)
  .create(client)

# snippet create-asset-acme-common
chain.assets.create()
  .setAlias('acme_common')
  .addRootXpub(asset_key.xpub)
  .setQuorum(1)
  .addTag('internal_rating', '1')
  .addDefinitionField('issuer', 'Acme Inc.')
  .addDefinitionField('type', 'security')
  .addDefinitionField('subtype', 'private')
  .addDefinitionField('class', 'common')
  .create(client)
# endsnippet

# snippet create-asset-acme-preferred
chain.assets.create()
  .setAlias('acme_preferred')
  .addRootXpub(asset_key.xpub)
  .setQuorum(1)
  .addTag('internal_rating', '2')
  .addDefinitionField('issuer', 'Acme Inc.')
  .addDefinitionField('type', 'security')
  .addDefinitionField('subtype', 'private')
  .addDefinitionField('class', 'perferred')
  .create(client)
# endsnippet

# snippet list-local-assets
localAssets = chain.assets.query
  .setFilter('is_local=$1')
  .addFilterParameter('yes')
  .execute(client)

while (localAssets.hasNext()) {
  Asset asset = localAssets.next()
  puts('Local asset: ' + asset.alias)
}
# endsnippet

# snippet list-private-preferred-securities
common = chain.assets.query
  .setFilter('definition.type=$1 AND definition.subtype=$2 AND definition.class=$3')
  .addFilterParameter('security')
  .addFilterParameter('private')
  .addFilterParameter('preferred')
  .execute(client)
# endsnippet

# snippet build-issue
issuanceTransaction = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('acme_common')
    .setAmount(1000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('acme_treasury')
    .setAssetAlias('acme_common')
    .setAmount(1000)
  ).build(client)
# endsnippet

# snippet sign-issue
signedIssuanceTransaction = signer.sign(issuanceTransaction)
# endsnippet

# snippet submit-issue
chain.transactions.submit(signedIssuanceTransaction)
# endsnippet

ControlProgram externalProgram = chain.accounts.create_control_program()
  .controlWithAccountByAlias('acme_treasury')
  .create(client)

# snippet external-issue
externalIssuance = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('acme_preferred')
    .setAmount(2000)
  ).addAction(new Transaction.Action.ControlWithProgram()
    .setControlProgram(externalProgram)
    .setAssetAlias('acme_preferred')
    .setAmount(2000)
  ).build(client)

chain.transactions.submit(signer.sign(externalIssuance))
# endsnippet

# snippet build-retire
retirementTransaction = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('acme_treasury')
    .setAssetAlias('acme_common')
    .setAmount(50)
  ).addAction(new Transaction.Action.Retire()
    .setAssetAlias('acme_common')
    .setAmount(50)
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
  .setFilter('inputs(action=$1 AND asset_alias=$2)')
  .addFilterParameter('issue')
  .addFilterParameter('acme_common')
  .execute(client)

while (acmeCommonIssuances.hasNext()) {
  tx = acmeCommonIssuances.next()
  puts('Acme Common issued in tx ' + tx.id)
}
# endsnippet

# snippet list-transfers
acmeCommonTransfers = chain.transactions.query
  .setFilter('inputs(action=$1 AND asset_alias=$2)')
  .addFilterParameter('spend')
  .addFilterParameter('acme_common')
  .execute(client)

while (acmeCommonTransfers.hasNext()) {
  tx = acmeCommonTransfers.next()
  puts('Acme Common transferred in tx ' + tx.id)
}
# endsnippet

# snippet list-retirements
acmeCommonRetirements = chain.transactions.query
  .setFilter('outputs(action=$1 AND asset_alias=$2)')
  .addFilterParameter('retire')
  .addFilterParameter('acme_common')
  .execute(client)

while (acmeCommonRetirements.hasNext()) {
  tx = acmeCommonRetirements.next()
  puts('Acme Common retired in tx ' + tx.id)
}
# endsnippet

# snippet list-acme-common-balance
acmeCommonBalances = chain.balances.query
  .setFilter('asset_alias=$1')
  .addFilterParameter('acme_common')
  .execute(client)

acmeCommonBalance = acmeCommonBalances.next()
puts('Total circulation of Acme Common: ' + acmeCommonBalance.amount)
# endsnippet

# snippet list-acme-balance
acmeAnyBalances = chain.balances.query
  .setFilter('asset_definition.issuer=$1')
  .addFilterParameter('Acme Inc.')
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
UnspentOutput.Items acmeCommonUnspentOutputs = new UnspentOutput.QueryBuilder()
  .setFilter('asset_alias=$1')
  .addFilterParameter('acme_common')
  .execute(client)

while (acmeCommonUnspentOutputs.hasNext()) {
  UnspentOutput utxo = acmeCommonUnspentOutputs.next()
  puts('Acme Common held in output ' + utxo.transactionId + ':' + utxo.position)
}
# endsnippet
