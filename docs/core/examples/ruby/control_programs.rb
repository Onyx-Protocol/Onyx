require 'chain'

chain = Chain::Client.new
setup(client)

# snippet create-control-program
ControlProgram aliceProgram = chain.accounts.create_control_program()
  .controlWithAccountByAlias('alice')
  .create(client)
# endsnippet

# snippet build-transaction
paymentToProgram = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(10)
  ).addAction(new Transaction.Action.ControlWithProgram()
    .setControlProgram(aliceProgram.controlProgram)
    .setAssetAlias('gold')
    .setAmount(10)
  ).build(client)

chain.transactions.submit(signer.sign(paymentToProgram))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(10)
  ).addAction(new Transaction.Action.Retire()
    .setAssetAlias('gold')
    .setAmount(10)
  ).build(client)

chain.transactions.submit(signer.sign(retirement))
# endsnippet
}

public static void setup(chain) throws Exception {
key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create()
  .setAlias('gold')
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .create(client)

chain.accounts.create()
  .setAlias('alice')
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .create(client)

chain.accounts.create()
  .setAlias('bob')
  .addRootXpub(key.xpub)
  .setQuorum(1)
  .create(client)

chain.transactions.submit(signer.sign(chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)
))
