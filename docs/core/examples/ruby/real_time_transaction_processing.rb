require 'chain'

chain = Chain::Client.new
setup(client)

# snippet processing-thread
new Thread(() -> {
    processingLoop(client)
}).start()
# endsnippet

# snippet issue
issuance = chain.transactions.build do |b|
  .addAction(new Transaction.Action.Issue()
    .setAssetAlias('gold')
    .setAmount(100)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(100)
  ).build(client)

chain.transactions.submit(signer.sign(issuance))
# endsnippet

sleep(1)

# snippet transfer
transfer = chain.transactions.build do |b|
  .addAction(new Transaction.Action.SpendFromAccount()
    .setAccountAlias('alice')
    .setAssetAlias('gold')
    .setAmount(50)
  ).addAction(new Transaction.Action.ControlWithAccount()
    .setAccountAlias('bob')
    .setAssetAlias('gold')
    .setAmount(50)
  ).build(client)

chain.transactions.submit(signer.sign(transfer))
# endsnippet

sleep(1)
System.exit(0)
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

# snippet create-feed
feed = chain.transaction_feeds.create
  client,
  'local-transactions',
  'is_local='yes''
)
# endsnippet
}

public static void processingLoop(chain) {
try {
  # snippet get-feed
  feed = chain.transaction_feeds.get(
    client,
    'local-transactions'
  )
  # endsnippet

  # snippet processing-loop
  while (true) {
    tx = feed.next(client)
    processTransaction(tx)
    feed.ack(client)
  }
  # endsnippet
} catch (Exception e) {
  throw new RuntimeException(e)
}

# snippet processor-method
public static void processTransaction(tx) {
  puts('New transaction at ' + tx.timestamp)
  puts('\tID: ' + tx.id)

  for (int i = 0 i < tx.inputs.size() i++) {
    input = tx.inputs.get(i)
    puts('\tInput ' + i)
    puts('\t\tType: ' + input.type)
    puts('\t\tAsset: ' + input.assetAlias)
    puts('\t\tAmount: ' + input.amount)
    puts('\t\tAccount: ' + input.accountAlias)
  }

  for (int i = 0 i < tx.outputs.size() i++) {
    output = tx.outputs.get(i)
    puts('\tOutput ' + i)
    puts('\t\tType: ' + output.type)
    puts('\t\tPurpose: ' + output.purpose)
    puts('\t\tAsset: ' + output.assetAlias)
    puts('\t\tAmount: ' + output.amount)
    puts('\t\tAccount: ' + output.accountAlias)
  }
}
# endsnippet
