require 'chain'

chain = Chain::Client.new
Client otherCoreClient = Chain::Client.new
setup(client, otherCoreClient)

# snippet issue-within-core
issuance = chain.transactions.build do |b|
  b.issue
    asset_alias: 'gold',
    amount: 1000,
  b.control_with_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 1000,
  ).build(client)

signedIssuance = signer.sign(issuance)

chain.transactions.submit(signedIssuance)
# endsnippet

# snippet create-bob-issue-program
bob_program = chain.accounts.create_control_program()
  alias: 'bob'
  .create(otherCoreClient)
# endsnippet

# snippet issue-to-bob-program
issuanceToProgram = chain.transactions.build do |b|
  b.issue
    asset_alias: 'gold',
    amount: 10,
  )b.control_with_program
    control_program: bob_program.controlProgram,
    asset_alias: 'gold',
    amount: 10,
  ).build(client)

signedIssuanceToProgram = signer.sign(issuanceToProgram)

chain.transactions.submit(signedIssuanceToProgram)
# endsnippet

# snippet pay-within-core
payment = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 10,
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 10,
  ).build(client)

signedPayment = signer.sign(payment)

chain.transactions.submit(signedPayment)
# endsnippet

# snippet create-bob-payment-program
bob_program = chain.accounts.create_control_program()
  alias: 'bob'
  .create(otherCoreClient)
# endsnippet

# snippet pay-between-cores
paymentToProgram = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 10,
  )b.control_with_program
    control_program: bob_program.controlProgram,
    asset_alias: 'gold',
    amount: 10,
  ).build(client)

signedPaymentToProgram = signer.sign(paymentToProgram)

chain.transactions.submit(signedPaymentToProgram)
# endsnippet

if (client.equals(otherCoreClient)) {
  # snippet multiasset-within-core
  multiAssetPayment = chain.transactions.build do |b|
    b.spend_from_account
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 10,
    )b.spend_from_account
      account_alias: 'alice',
      asset_alias: 'silver',
      amount: 20,
    b.control_with_account
      account_alias: 'bob',
      asset_alias: 'gold',
      amount: 10,
    b.control_with_account
      account_alias: 'bob',
      asset_alias: 'silver',
      amount: 20,
    ).build(client)

  signedMultiAssetPayment = signer.sign(multiAssetPayment)

  chain.transactions.submit(signedMultiAssetPayment)
  # endsnippet
}

# snippet create-bob-multiasset-program
bob_program = chain.accounts.create_control_program()
  alias: 'bob'
  .create(otherCoreClient)
# endsnippet

# snippet multiasset-between-cores
multiAssetToProgram = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 10,
  )b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'silver',
    amount: 20,
  )b.control_with_program
    control_program: bob_program.controlProgram,
    asset_alias: 'gold',
    amount: 10,
  )b.control_with_program
    control_program: bob_program.controlProgram,
    asset_alias: 'silver',
    amount: 20,
  ).build(client)

signedMultiAssetToProgram = signer.sign(multiAssetToProgram)

chain.transactions.submit(signedMultiAssetToProgram)
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 50,
  ).addAction(new Transaction.Action.Retire()
    asset_alias: 'gold',
    amount: 50,
  ).build(client)

signedRetirement = signer.sign(retirement)

chain.transactions.submit(signedRetirement)
# endsnippet
}

public static void setup(chain, Client otherCoreClient) throws Exception {
alice_key = chain.mock_hsm.keys.create
signer.add_key(alice_key, chain.mock_hsm.signer_conn)

bob_key = MockHsm.Key.create(otherCoreClient)
signer.add_key(bob_key, MockHsm.getSignerClient(otherCoreClient))

chain.assets.create(
  alias: 'gold',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
)

chain.assets.create(
  alias: 'silver',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'alice',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
)

chain.accounts.create(
  alias: 'bob',
  root_xpubs: [bob_key.xpub],
  quorum: 1,
  .create(otherCoreClient)

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  b.issue
    asset_alias: 'silver',
    amount: 1000,
  b.control_with_account
    account_alias: 'alice',
    asset_alias: 'silver',
    amount: 1000,
  ).build(client)
))
