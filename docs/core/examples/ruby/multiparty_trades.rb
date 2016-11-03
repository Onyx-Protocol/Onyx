require 'chain'

# This demo is written to run on either one or two cores. Simply provide
# different URLs to the following clients for the two-core version.
Client aliceCore = Chain::Client.new
Client bobCore = Chain::Client.new

alice_dollar_key = MockHsm.Key.create(aliceCore)
signer.add_key(alice_dollar_key, MockHsm.getSignerClient(aliceCore))

bob_buck_key = MockHsm.Key.create(bobCore)
signer.add_key(bob_buck_key, MockHsm.getSignerClient(bobCore))

alice_key = MockHsm.Key.create(aliceCore)
signer.add_key(alice_key, MockHsm.getSignerClient(aliceCore))

bob_key = MockHsm.Key.create(bobCore)
signer.add_key(bob_key, MockHsm.getSignerClient(bobCore))

Asset aliceDollar = chain.assets.create()
  .setAlias('aliceDollar')
  .addRootXpub(alice_dollar_key.xpub)
  .setQuorum(1)
  .create(aliceCore)

Asset bobBuck = chain.assets.create()
  .setAlias('bobBuck')
  .addRootXpub(bob_buck_key.xpub)
  .setQuorum(1)
  .create(bobCore)

Account alice = chain.accounts.create()
  .setAlias('alice')
  .addRootXpub(alice_key.xpub)
  .setQuorum(1)
  .create(aliceCore)

Account bob = chain.accounts.create()
  .setAlias('bob')
  .addRootXpub(bob_key.xpub)
  .setQuorum(1)
  .create(bobCore)

chain.transactions.submit(aliceCore, signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('aliceDollar')
    .setAmount(1000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('aliceDollar')
    .setAmount(1000)
  ).build(aliceCore)
))

chain.transactions.submit(bobCore, signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('bobBuck')
    .setAmount(1000)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('bobBuck')
    .setAmount(1000)
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
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('aliceDollar')
    .setAmount(50)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('bobBuck')
    .setAmount(100)
  ).addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('bob')
    .setAssetAlias('bobBuck')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('aliceDollar')
    .setAmount(50)
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
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('aliceDollar')
    .setAmount(50)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetId(bobBuckAssetId)
    .setAmount(100)
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
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('bob')
    .setAssetAlias('bobBuck')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetId(aliceDollarAssetId)
    .setAmount(50)
  ).build(bobCore)
# endsnippet

# snippet sign-trade-bob
bobTradeSigned = signer.sign(bobTrade)
# endsnippet

# snippet submit-trade-bob
chain.transactions.submit(bobCore, bobTradeSigned)
# endsnippet
