require 'chain'

# This demo is written to run on either one or two cores. Simply provide
# different URLs to the following clients for the two-core version.
alice_core = Chain::Client.new
bob_core = Chain::Client.new

alice_signer = Chain::HSMSigner.new
bob_signer = Chain::HSMSigner.new

alice_dollar_key = alice_core.mock_hsm.keys.create
alice_signer.add_key(alice_dollar_key, alice_core.mock_hsm.signer_conn)

bob_buck_key = bob_core.mock_hsm.keys.create
bob_signer.add_key(bob_buck_key, bob_core.mock_hsm.signer_conn)

alice_key = alice_core.mock_hsm.keys.create
alice_signer.add_key(alice_key, alice_core.mock_hsm.signer_conn)

bob_key = bob_core.mock_hsm.keys.create
bob_signer.add_key(bob_key, bob_core.mock_hsm.signer_conn)

alice_dollar = alice_core.assets.create(
  alias: 'alice_dollar',
  root_xpubs: [alice_dollar_key.xpub],
  quorum: 1,
)

bob_buck = bob_core.assets.create(
  alias: 'bob_buck',
  root_xpubs: [bob_buck_key.xpub],
  quorum: 1,
)

alice = alice_core.accounts.create(
  alias: 'alice',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
)

bob = bob_core.accounts.create(
  alias: 'bob',
  root_xpubs: [bob_key.xpub],
  quorum: 1,
)

tx = alice_core.transactions.build do |b|
  b.issue asset_alias: 'alice_dollar', amount: 1000
  b.control_with_account account_alias: 'alice', asset_alias: 'alice_dollar', amount: 1000
end

alice_core.transactions.submit(alice_signer.sign(tx))

tx = bob_core.transactions.build do |b|
  b.issue asset_alias: 'bob_buck', amount: 1000
  b.control_with_account account_alias: 'bob', asset_alias: 'bob_buck', amount: 1000
end

bob_core.transactions.submit(bob_signer.sign(tx))

if alice_core.opts[:url] == bob_core.opts[:url]
  chain = alice_core
  signer = alice_signer
  signer.add_key(bob_key, chain.mock_hsm.signer_conn)

  # SAME-CORE TRADE

  # snippet same-core-trade
  trade = chain.transactions.build do |b|
    b.spend_from_account account_alias: 'alice', asset_alias: 'alice_dollar', amount: 50
    b.control_with_account account_alias: 'alice', asset_alias: 'bob_buck', amount: 100
    b.spend_from_account account_alias: 'bob', asset_alias: 'bob_buck', amount: 100
    b.control_with_account account_alias: 'bob', asset_alias: 'alice_dollar', amount: 50
  end

  chain.transactions.submit(signer.sign(trade))
  # endsnippet
else
  # CROSS-CORE TRADE

  alice_dollar_asset_id = alice_dollar.id
  bob_buck_asset_id = bob_buck.id

  # snippet build-trade-alice
  alice_trade = alice_core.transactions.build do |b|
    b.spend_from_account account_alias: 'alice', asset_alias: 'alice_dollar', amount: 50
    b.control_with_account account_alias: 'alice', asset_id: bob_buck_asset_id, amount: 100
  end
  # endsnippet

  # snippet sign-trade-alice
  alice_trade_signed = alice_signer.sign(alice_trade.allow_additional_actions)
  # endsnippet

  # snippet base-transaction-alice
  base_tx_from_alice = alice_trade_signed.raw_transaction
  # endsnippet

  # snippet build-trade-bob
  bob_trade = bob_core.transactions.build do |b|
    b.base_transaction base_tx_from_alice
    b.spend_from_account account_alias: 'bob', asset_alias: 'bob_buck', amount: 100
    b.control_with_account account_alias: 'bob', asset_id: alice_dollar_asset_id, amount: 50
  end
  # endsnippet

  # snippet sign-trade-bob
  bob_trade_signed = bob_signer.sign(bob_trade)
  # endsnippet

  # snippet submit-trade-bob
  bob_core.transactions.submit(bob_trade_signed)
  # endsnippet
end
