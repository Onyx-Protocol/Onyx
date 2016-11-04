require 'chain'

chain = Chain::Client.new
setup(client)

# snippet list-alice-transactions
aliceTransactions = chain.transactions.query
  filter: 'inputs(account_alias=$1, OR outputs(account_alias=$1)')
  filter_params: ['alice'],
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
  filter: 'is_local=$1',
  filter_params: ['yes'],
  .execute(client)

while (localTransactions.hasNext()) {
  transaction = localTransactions.next()
  puts('Local transaction ' + transaction.id)
}
# endsnippet

# snippet list-local-assets
localAssets = chain.assets.query
  filter: 'is_local=$1',
  filter_params: ['yes'],
  .execute(client)

while (localAssets.hasNext()) {
  Asset asset = localAssets.next()
  puts('Local asset ' + asset.id + ' (' + asset.alias + ')')
}
# endsnippet

# snippet list-usd-assets
usdAssets = chain.assets.query
  filter: 'definition.currency=$1',
  filter_params: ['USD'],
  .execute(client)

while (usdAssets.hasNext()) {
  Asset asset = usdAssets.next()
  puts('USD asset ' + asset.id + ' (' + asset.alias + ')')
}
# endsnippet

# snippet list-checking-accounts
checkingAccounts = chain.accounts.query
  filter: 'tags.type=$1',
  filter_params: ['checking'],
  .execute(client)

while (checkingAccounts.hasNext()) {
  Account account = checkingAccounts.next()
  puts('Checking account ' + account.id + ' (' + account.alias + ')')
}
# endsnippet

# snippet list-alice-unspents
aliceUnspentOuputs = chain.unspent_outputs.query()
  filter: 'account_alias=$1',
  filter_params: ['alice'],
  .execute(client)

while (aliceUnspentOuputs.hasNext()) {
  UnspentOutput utxo = aliceUnspentOuputs.next()
  puts('Alice\'s unspent output: ' + utxo.amount + ' ' + utxo.assetAlias)
}
# endsnippet

# snippet account-balance
bank1Balances = chain.balances.query
  filter: 'account_alias=$1',
  filter_params: ['bank1'],
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
  filter: 'asset_alias=$1',
  filter_params: ['bank1_usd_iou'],
  .execute(client)

bank1UsdIouCirculation = bank1UsdIouBalances.next()
puts('Total circulation of Bank 1 USD IOU: ' + bank1UsdIouCirculation.amount)
# endsnippet

# snippet account-balance-sum-by-currency
bank1CurrencyBalances = chain.balances.query
  filter: 'account_alias=$1',
  filter_params: ['bank1'],
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
  type: 'checking',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'bob',
  root_xpubs: [key.xpub],
  quorum: 1,
)

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  b.issue
    asset_alias: 'gold',
    amount: 1000,)
  b.issue
    asset_alias: 'silver',
    amount: 1000,)
  .addAction(new Transaction.Action.ControlWithAccount()
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 1000,)
  .addAction(new Transaction.Action.ControlWithAccount()
    account_alias: 'bob',
    asset_alias: 'silver',
    amount: 1000,)
  .build(client)))

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 10,)
  b.spend_from_account
    account_alias: 'bob',
    asset_alias: 'silver',
    amount: 10,)
  .addAction(new Transaction.Action.ControlWithAccount()
    account_alias: 'alice',
    asset_alias: 'silver',
    amount: 10,)
  .addAction(new Transaction.Action.ControlWithAccount()
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 10,)
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
  b.issue
    asset_alias: 'bank1_usd_iou',
    amount: 2000000,
  )b.issue
    asset_alias: 'bank2_usd_iou',
    amount: 2000000,
  )b.issue
    asset_alias: 'bank1_euro_iou',
    amount: 2000000,
  b.control_with_account
    account_alias: 'bank1',
    asset_alias: 'bank1_usd_iou',
    amount: 1000000,
  b.control_with_account
    account_alias: 'bank1',
    asset_alias: 'bank1_euro_iou',
    amount: 1000000,
  b.control_with_account
    account_alias: 'bank1',
    asset_alias: 'bank2_usd_iou',
    amount: 1000000,
  b.control_with_account
    account_alias: 'bank2',
    asset_alias: 'bank1_usd_iou',
    amount: 1000000,
  b.control_with_account
    account_alias: 'bank2',
    asset_alias: 'bank1_euro_iou',
    amount: 1000000,
  b.control_with_account
    account_alias: 'bank2',
    asset_alias: 'bank2_usd_iou',
    amount: 1000000,
  ).build(client)
))
