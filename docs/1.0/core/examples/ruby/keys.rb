require 'chain'

chain = Chain::Client.new

# snippet create-key
key = chain.mock_hsm.keys.create
# endsnippet

# snippet signer-add-key
signer = Chain::HSMSigner.new # Holds multiple keys.
signer.add_key(key, chain.mock_hsm.signer_conn)
# endsnippet

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

unsigned = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
end

# snippet sign-transaction
signed = signer.sign(unsigned)
# endsnippet
