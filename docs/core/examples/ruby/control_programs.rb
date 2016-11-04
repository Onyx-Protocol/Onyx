require 'chain'

chain = Chain::Client.new
setup(client)

# snippet create-control-program
aliceProgram = chain.accounts.create_control_program()
  alias: 'alice'
)
# endsnippet

# snippet build-transaction
paymentToProgram = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 10,
  b.control_with_program
    control_program: aliceProgram.controlProgram,
    asset_alias: 'gold',
    amount: 10,
  ).build(client)

chain.transactions.submit(signer.sign(paymentToProgram))
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account
    account_alias: 'alice',
    asset_alias: 'gold',
    amount: 10,
  b.retire
    asset_alias: 'gold',
    amount: 10,
  ).build(client)

chain.transactions.submit(signer.sign(retirement))
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

chain.accounts.create(
  alias: 'alice',
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
    amount: 100,
  b.control_with_account
    account_alias: 'bob',
    asset_alias: 'gold',
    amount: 100,
  ).build(client)
))
