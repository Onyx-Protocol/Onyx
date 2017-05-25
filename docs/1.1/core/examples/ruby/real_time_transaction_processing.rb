require "chain"

chain = Chain::Client.new
$chain = chain # global alias for thread funcs
signer = Chain::HSMSigner.new

key = chain.mock_hsm.keys.create
signer.add_key(key, chain.mock_hsm.signer_conn)

chain.assets.create(alias: 'gold', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'alice', root_xpubs: [key.xpub], quorum: 1)
chain.accounts.create(alias: 'bob', root_xpubs: [key.xpub], quorum: 1)

# snippet create-feed
chain.transaction_feeds.create(
  alias: 'local-transactions',
  filter: "is_local='yes'"
)
# endsnippet

def run_processing_loop
  chain = $chain

  # snippet get-feed
  feed = chain.transaction_feeds.get(alias: 'local-transactions')
  # endsnippet

  # snippet processing-loop
  feed.consume do |tx|
    process_transaction(tx)
    feed.ack
  end
  # endsnippet
end

# snippet processor-method
def process_transaction(tx)
  puts "New transaction at #{tx.timestamp}"
  puts "\tID: #{tx.id}"

  tx.inputs.each_with_index do |input, index|
    puts "\tInput #{index}"
    puts "\t\tType: #{input.type}"
    puts "\t\tAsset: #{input.asset_alias}"
    puts "\t\tAmount: #{input.amount}"
    puts "\t\tAccount: #{input.account_alias}"
  end

  tx.outputs.each_with_index do |output, index|
    puts "\tOutput #{index}"
    puts "\t\tType: #{output.type}"
    puts "\t\tPurpose: #{output.purpose}"
    puts "\t\tAsset: #{output.asset_alias}"
    puts "\t\tAmount: #{output.amount}"
    puts "\t\tAccount: #{output.account_alias}"
  end
end
# endsnippet

# snippet processing-thread
Thread.new { run_processing_loop }
# endsnippet

# snippet issue
issuance = chain.transactions.build do |b|
  b.issue asset_alias: 'gold', amount: 100
  b.control_with_account account_alias: 'alice', asset_alias: 'gold', amount: 100
end

chain.transactions.submit(signer.sign(issuance))
# endsnippet

sleep(0.25)

# snippet transfer
transfer = chain.transactions.build do |b|
  b.spend_from_account account_alias: 'alice', asset_alias: 'gold', amount: 50
  b.control_with_account account_alias: 'bob', asset_alias: 'gold', amount: 50
end

chain.transactions.submit(signer.sign(transfer))
# endsnippet

sleep(0.25)
exit
