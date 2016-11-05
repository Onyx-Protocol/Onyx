require 'chain'

# This demo is written to run on either one or two cores. Simply provide
# different URLs to the following clients for the two-core version.
chain = Chain::Client.new
other_core = Chain::Client.new

signer = Chain::HSMSigner.new

alice_key = chain.mock_hsm.keys.create
signer.add_key(alice_key, chain.mock_hsm.signer_conn)

bob_key = other_core.mock_hsm.keys.create

chain.assets.create(alias: 'gold', root_xpubs: [alice_key.xpub], quorum: 1)
chain.assets.create(alias: 'silver', root_xpubs: [alice_key.xpub], quorum: 1)
chain.accounts.create(alias: 'alice', root_xpubs: [alice_key.xpub], quorum: 1)
other_core.accounts.create(alias: 'bob', root_xpubs: [bob_key.xpub], quorum: 1)

chain.transactions.submit(signer.sign(chain.transactions.build { |b|
  b.issue asset_alias: 'silver', amount: 1000
  b.control_with_account account_alias: 'alice', asset_alias: 'silver', amount: 1000
}))

# snippet issue-within-core
issuance = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 1000
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 1000
end

signed_issuance = signer.sign(issuance)

chain.transactions.submit(signed_issuance)
# endsnippet

# snippet create-bob-issue-program
bob_program = other_core.accounts.create_control_program(
  alias: 'bob'
).control_program
# endsnippet

# snippet issue-to-bob-program
issuance_to_program = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 10
  b.control_with_program control_program: bob_program, asset_alias: 'gold', amount: 10
end

signed_issuance_to_program = signer.sign(issuance_to_program)

chain.transactions.submit(signed_issuance_to_program)
# endsnippet

if (chain.opts[:url] == other_core.opts[:url])
  # snippet pay-within-core
  payment = chain.transactions.build do |b|
    b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
    b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 10
  end

  signed_payment = signer.sign(payment)

  chain.transactions.submit(signed_payment)
  # endsnippet
end

# snippet create-bob-payment-program
bob_program = other_core.accounts.create_control_program(
  alias: 'bob'
).control_program
# endsnippet

# snippet pay-between-cores
payment_to_program = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.control_with_program control_program: bob_program, asset_alias: 'gold', amount: 10
end

signed_payment_to_program = signer.sign(payment_to_program)

chain.transactions.submit(signed_payment_to_program)
# endsnippet

if (chain.opts[:url] == other_core.opts[:url])
  # snippet multiasset-within-core
  multi_asset_payment = chain.transactions.build do |b|
    b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
    b.spend_from_account account_alias: 'alice', asset_alias: 'silver', amount: 20
    b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 10
    b.control_with_account account_alias: 'bob', asset_alias: 'silver', amount: 20
  end

  signed_multi_asset_payment = signer.sign(multi_asset_payment)

  chain.transactions.submit(signed_multi_asset_payment)
  # endsnippet
end

# snippet create-bob-multiasset-program
bob_program = other_core.accounts.create_control_program(
  alias: 'bob'
).control_program
# endsnippet

# snippet multiasset-between-cores
multi_asset_to_program = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 10
  b.spend_from_account account_alias: 'alice', asset_alias: 'silver', amount: 20
  b.control_with_program control_program: bob_program, asset_alias: 'gold', amount: 10
  b.control_with_program control_program: bob_program, asset_alias: 'silver', amount: 20
end

signed_multi_asset_to_program = signer.sign(multi_asset_to_program)

chain.transactions.submit(signed_multi_asset_to_program)
# endsnippet

# snippet retire
retirement = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 50
  b.retire asset_alias: 'gold', amount: 50
end

signed_retirement = signer.sign(retirement)

chain.transactions.submit(signed_retirement)
# endsnippet
