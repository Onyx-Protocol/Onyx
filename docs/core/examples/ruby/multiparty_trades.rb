require 'chain'

# This demo is written to run on either one or two cores. Simply provide
# different URLs to the following clients for the two-core version.
Client aliceCore = Chain::Client.new
Client bobCore = Chain::Client.new
signer = Chain::HSMSigner.new

alice_dollar_key = MockHsm.Key.create(aliceCore)
signer.add_key(alice_dollar_key, MockHsm.getSignerClient(aliceCore))

bob_buck_key = MockHsm.Key.create(bobCore)
signer.add_key(bob_buck_key, MockHsm.getSignerClient(bobCore))

alice_key = MockHsm.Key.create(aliceCore)
signer.add_key(alice_key, MockHsm.getSignerClient(aliceCore))

bob_key = MockHsm.Key.create(bobCore)
signer.add_key(bob_key, MockHsm.getSignerClient(bobCore))

Asset aliceDollar = chain.assets.create(
  alias: 'aliceDollar',
  root_xpubs: [alice_dollar_key.xpub],
  quorum: 1,
  .create(aliceCore)

Asset bobBuck = chain.assets.create(
  alias: 'bobBuck',
  root_xpubs: [bob_buck_key.xpub],
  quorum: 1,
  .create(bobCore)

Account alice = chain.accounts.create(
  alias: 'alice',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
  .create(aliceCore)

Account bob = chain.accounts.create(
  alias: 'bob',
  root_xpubs: [bob_key.xpub],
  quorum: 1,
  .create(bobCore)

chain.transactions.submit(aliceCore, signer.sign(chain.transactions.build do |b|
  b.issue
    asset_alias: 'aliceDollar',
    amount: 1000,
  b.control_with_account
    account_alias: 'alice',
    asset_alias: 'aliceDollar',
    amount: 1000,
  ).build(aliceCore)
))

chain.transactions.submit(bobCore, signer.sign(chain.transactions.build do |b|
  b.issue
    asset_alias: 'bobBuck',
    amount: 1000,
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'bobBuck',
    amount: 1000,
  ).build(bobCore)
))

if (aliceCore.equals(bobCore)) {
  sameCore(aliceCore)
}

crossCore(aliceCore, bobCore, alice, bob, aliceDollar.id, bobBuck.id)
}

public static void sameCore(chain) throws Exception {
# snippet same-core-trade
trade = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'aliceDollar',
    amount: 50,
  b.control_with_account
    account_alias: 'alice',
    asset_alias: 'bobBuck',
    amount: 100,
  b.spend_from_account
    account_alias: 'bob',
    asset_alias: 'bobBuck',
    amount: 100,
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'aliceDollar',
    amount: 50,
  ).build(client)

chain.transactions.submit(signer.sign(trade))
# endsnippet
}

public static void crossCore(
Client aliceCore, Client bobCore,
Account alice, Account bob,
String aliceDollarAssetId, String bobBuckAssetId
) throws Exception {
# snippet build-trade-alice
aliceTrade = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'aliceDollar',
    amount: 50,
  b.control_with_account
    account_alias: 'alice',
    .setAssetId(bobBuckAssetId)
    amount: 100,
  ).build(aliceCore)
# endsnippet

# snippet sign-trade-alice
aliceTradeSigned = signer.sign(aliceTrade.allowAdditionalActions())
# endsnippet

# snippet base-transaction-alice
String baseTransactionFromAlice = aliceTradeSigned.rawTransaction
# endsnippet

# snippet build-trade-bob
bobTrade = chain.transactions.build do |b|
  .setBaseTransaction(baseTransactionFromAlice)
  b.spend_from_account
    account_alias: 'bob',
    asset_alias: 'bobBuck',
    amount: 100,
  b.control_with_account
    account_alias: 'bob',
    .setAssetId(aliceDollarAssetId)
    amount: 50,
  ).build(bobCore)
# endsnippet

# snippet sign-trade-bob
bobTradeSigned = signer.sign(bobTrade)
# endsnippet

# snippet submit-trade-bob
chain.transactions.submit(bobCore, bobTradeSigned)
# endsnippet
