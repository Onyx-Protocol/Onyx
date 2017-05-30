require 'chain'

chain = Chain::Client.new
signer = Chain::HSMSigner.new

asset_key = chain.mock_hsm.keys.create
signer.add_key(asset_key, chain.mock_hsm.signer_conn)

alice_key = chain.mock_hsm.keys.create
signer.add_key(alice_key, chain.mock_hsm.signer_conn)

bob_key = chain.mock_hsm.keys.create
signer.add_key(bob_key, chain.mock_hsm.signer_conn)

# snippet create-accounts-with-tags
chain.accounts.create(
  alias: 'alice',
  root_xpubs: [alice_key.xpub],
  quorum: 1,
  tags: {
    type: 'checking',
    first_name: 'Alice',
    last_name: 'Jones',
    user_id: '12345',
    status: 'enabled'
  }
)

chain.accounts.create(
  alias: 'bob',
  root_xpubs: [bob_key.xpub],
  quorum: 1,
  tags: {
    type: 'checking',
    first_name: 'Bob',
    last_name: 'Smith',
    user_id: '67890',
    status: 'enabled'
  }
)
#endsnippet

# snippet create-asset-with-tags-and-definition
chain.assets.create(
  alias: 'acme_bond',
  root_xpubs: [asset_key.xpub],
  quorum: 1,
  tag: {
    internal_rating: 'B',
  },
  definition: {
    type: 'security',
    sub_type: 'corporate-bond',
    entity: 'Acme Inc.',
    maturity: '2016-09-01T18:24:47+00:00'
  }
)
# endsnippet


# snippet build-tx-with-tx-ref-data
tx_with_ref_data = chain.transactions.build do |b|
  b.issue asset_alias: 'acme_bond', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'acme_bond', amount: 100
  b.transaction_reference_data external_reference: '12345'
end
# endsnippet

chain.transactions.submit(signer.sign(tx_with_ref_data))

# snippet build-tx-with-action-ref-data
tx_with_action_ref_data = chain.transactions.build do |b|
  b.issue asset_alias: 'acme_bond', amount: 100
  b.retire asset_alias: 'acme_bond', amount: 100,
    reference_data: {external_reference: '12345'}
end
# endsnippet

chain.transactions.submit(signer.sign(tx_with_action_ref_data))
