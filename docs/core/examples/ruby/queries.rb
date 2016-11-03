require 'chain'

chain = Chain::Client.new
setup(client)

# snippet list-alice-transactions
aliceTransactions = chain.transactions.query
  .setFilter('inputs(account_alias=$1) OR outputs(account_alias=$1)')
  .addFilterParameter('alice')
  .execute(client)

while (aliceTransactions.hasNext()) {
  transaction = aliceTransactions.next()
  puts('Alice\'s transaction ' + transaction.id)
  for (input: transaction.inputs) {
    if (input.accountAlias != null && input.accountAlias.equals('alice')) {
      puts('  -' + input.amount + ' ' + input.assetAlias)
    }
  }
  for (output: transaction.outputs) {
    if (output.accountAlias != null && output.accountAlias.equals('alice')) {
      puts('  +' + output.amount + ' ' + output.assetAlias)
    }
  }
}
# endsnippet

# snippet list-local-transactions
localTransactions = chain.transactions.query
  .setFilter('is_local=$1')
  .addFilterParameter('yes')
  .execute(client)

while (localTransactions.hasNext()) {
  transaction = localTransactions.next()
  puts('Local transaction ' + transaction.id)
}
# endsnippet

# snippet list-local-assets
localAssets = chain.assets.query
  .setFilter('is_local=$1')
  .addFilterParameter('yes')
  .execute(client)

while (localAssets.hasNext()) {
  Asset asset = localAssets.next()
  puts('Local asset ' + asset.id + ' (' + asset.alias + ')')
}
# endsnippet

# snippet list-usd-assets
usdAssets = chain.assets.query
  .setFilter('definition.currency=$1')
  .addFilterParameter('USD')
  .execute(client)

while (usdAssets.hasNext()) {
  Asset asset = usdAssets.next()
  puts('USD asset ' + asset.id + ' (' + asset.alias + ')')
}
# endsnippet

# snippet list-checking-accounts
checkingAccounts = chain.accounts.query
  .setFilter('tags.type=$1')
  .addFilterParameter('checking')
  .execute(client)

while (checkingAccounts.hasNext()) {
  Account account = checkingAccounts.next()
  puts('Checking account ' + account.id + ' (' + account.alias + ')')
}
# endsnippet

# snippet list-alice-unspents
UnspentOutput.Items aliceUnspentOuputs = new UnspentOutput.QueryBuilder()
  .setFilter('account_alias=$1')
  .addFilterParameter('alice')
  .execute(client)

while (aliceUnspentOuputs.hasNext()) {
  UnspentOutput utxo = aliceUnspentOuputs.next()
  puts('Alice\'s unspent output: ' + utxo.amount + ' ' + utxo.assetAlias)
}
# endsnippet

# snippet account-balance
bank1Balances = chain.balances.query
  .setFilter('account_alias=$1')
  .addFilterParameter('bank1')
  .execute(client)

while (bank1Balances.hasNext()) {
  b = bank1Balances.next()
  puts(
    'Bank 1 balance of ' + b.sumBy.get('asset_alias') +
    ': ' + b.amount
  )
}
# endsnippet

# snippet usd-iou-circulation
bank1UsdIouBalances = chain.balances.query
  .setFilter('asset_alias=$1')
  .addFilterParameter('bank1_usd_iou')
  .execute(client)

bank1UsdIouCirculation = bank1UsdIouBalances.next()
puts('Total circulation of Bank 1 USD IOU: ' + bank1UsdIouCirculation.amount)
# endsnippet

# snippet account-balance-sum-by-currency
bank1CurrencyBalances = chain.balances.query
  .setFilter('account_alias=$1')
  .addFilterParameter('bank1')
  .setSumBy(Arrays.asList('asset_definition.currency'))
  .execute(client)

while (bank1CurrencyBalances.hasNext()) {
  b = bank1CurrencyBalances.next()
  puts(
    'Bank 1 balance of ' + b.sumBy.get('asset_definition.currency') +
    '-denominated currencies : ' + b.amount
  )
}
# endsnippet
}

public static void setup(chain) throws Exception {
key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create(
  alias: 'gold',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.assets.create(
  alias: 'silver',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'alice',
  .addTag('type', 'checking')
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'bob',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(1000))
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('silver')
    .setAmount(1000))
  .addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(1000))
  .addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('silver')
    .setAmount(1000))
  .build(client)))

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(10))
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('bob')
    .setAssetAlias('silver')
    .setAmount(10))
  .addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('silver')
    .setAmount(10))
  .addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(10))
  .build(client)))

chain.assets.create(
  alias: 'bank1_usd_iou',
  root_xpubs: [key.xpub],
  quorum: 1,
  .addDefinitionField('currency', 'USD')
)

chain.assets.create(
  alias: 'bank1_euro_iou',
  root_xpubs: [key.xpub],
  quorum: 1,
  .addDefinitionField('currency', 'Euro')
)

chain.assets.create(
  alias: 'bank2_usd_iou',
  root_xpubs: [key.xpub],
  quorum: 1,
  .addDefinitionField('currency', 'USD')
)

chain.accounts.create(
  alias: 'bank1',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'bank2',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('bank1_usd_iou')
    .setAmount(2000000)
  ).addAction(new Transaction.Action.Issue()
    .setAssetAlias('bank2_usd_iou')
    .setAmount(2000000)
  ).addAction(new Transaction.Action.Issue()
    .setAssetAlias('bank1_euro_iou')
    .setAmount(2000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank1')
    .setAssetAlias('bank1_usd_iou')
    .setAmount(1000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank1')
    .setAssetAlias('bank1_euro_iou')
    .setAmount(1000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank1')
    .setAssetAlias('bank2_usd_iou')
    .setAmount(1000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank2')
    .setAssetAlias('bank1_usd_iou')
    .setAmount(1000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank2')
    .setAssetAlias('bank1_euro_iou')
    .setAmount(1000000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bank2')
    .setAssetAlias('bank2_usd_iou')
    .setAmount(1000000)
  ).build(client)
))
