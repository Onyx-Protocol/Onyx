require 'chain'

chain = Chain::Client.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

# snippet asset-builders
List<Asset.Builder> assetBuilders = Arrays.asList(
  chain.assets.create(
    alias: 'gold',
    root_xpubs: [key.xpub],
    quorum: 1,,
  chain.assets.create(
    alias: 'silver',
    root_xpubs: [key.xpub],
    quorum: 1,,
  chain.assets.create(
    alias: 'bronze',
    root_xpubs: [key.xpub],
    quorum: [0],
)
# endsnippet

# snippet asset-create-batch
BatchResponse<Asset> assetBatch = Asset.createBatch(client, assetBuilders)
# endsnippet

# snippet asset-create-handle-errors
for (int i = 0 i < assetBatch.size() i++) {
  if (assetBatch.isError(i)) {
    APIException error = assetBatch.errorsByIndex().get(i)
    puts('asset ' + i + ' error: ' + error)
  } else {
    Asset asset = assetBatch.successesByIndex().get(i)
    puts('asset ' + i + ' created, ID: ' + asset.id)
  }
}
# endsnippet

# snippet nondeterministic-errors
assetBuilders = Arrays.asList(
  chain.assets.create(
    alias: 'platinum',
    root_xpubs: [key.xpub],
    quorum: 1,,
  chain.assets.create(
    alias: 'platinum',
    root_xpubs: [key.xpub],
    quorum: 1,,
  chain.assets.create(
    alias: 'platinum',
    root_xpubs: [key.xpub],
    quorum: 1,
)
# endsnippet

assetBatch = Asset.createBatch(client, assetBuilders)

for (int i = 0 i < assetBatch.size() i++) {
  if (assetBatch.isError(i)) {
    APIException error = assetBatch.errorsByIndex().get(i)
    puts('asset ' + i + ' error: ' + error)
  } else {
    Asset asset = assetBatch.successesByIndex().get(i)
    puts('asset ' + i + ' created, ID: ' + asset.id)
  }
}

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

# snippet batch-build-builders
List<Transaction.Builder> txBuilders = Arrays.asList(
  chain.transactions.build do |b|
    b.issue
      asset_alias: 'gold',
      amount: 100,
    b.control_with_account
      account_alias: 'alice',
      asset_alias: 'gold',
      amount: 100,
    ),
  chain.transactions.build do |b|
    b.issue
      asset_alias: 'not-a-real-asset',
      amount: 100,
    b.control_with_account
      account_alias: 'alice',
      asset_alias: 'not-a-real-asset',
      amount: 100,
    ),
  chain.transactions.build do |b|
    b.issue
      asset_alias: 'silver',
      amount: 100,
    b.control_with_account
      account_alias: 'alice',
      asset_alias: 'silver',
      amount: 100,
    )
)
# endsnippet

# snippet batch-build-handle-errors
BatchResponse<Transaction.Template> buildTxBatch = Transaction.buildBatch(client, txBuilders)

for(Map.Entry<Integer, APIException> err : buildTxBatch.errorsByIndex().entrySet()) {
  puts('Error building transaction ' + err.getKey() + ': ' + err.getValue())
}
# endsnippet

# snippet batch-sign
BatchResponse<Transaction.Template> signTxBatch = signer.signBatch(buildTxBatch.successes())

for(Map.Entry<Integer, APIException> err : signTxBatch.errorsByIndex().entrySet()) {
  puts('Error signing transaction ' + err.getKey() + ': ' + err.getValue())
}
# endsnippet

# snippet batch-submit
BatchResponse<Transaction.SubmitResponse> submitTxBatch = Transaction.submitBatch(client, signTxBatch.successes())

for(Map.Entry<Integer, APIException> err : submitTxBatch.errorsByIndex().entrySet()) {
  puts('Error submitting transaction ' + err.getKey() + ': ' + err.getValue())
}

for(Map.Entry<Integer, Transaction.SubmitResponse> success : submitTxBatch.successesByIndex().entrySet()) {
  puts('' + success.getKey() + ' submitted, ID: ' + success.getValue().id)
}
# endsnippet
