
TODO:

- Tx = {txversion, mintime, maxtime, [entry] }
- Entry = {content, witness}
- TxID = Hash(txversion, mintime, maxtime, [{entry.entrytype, entry.content}] )
- TxWitHash = Hash(TxID, [{entry.witness}] )
- Content = {entrytype, ...}
- Content = {0, refdata_hash}
- Content = {1, entry_type, ...}
- Content = {1, "issue", ...}
- Content = {1, "input", outpoint}
- Content = {1, "output", ...}
- Content = {1, "retire", ...}
- remove OP_FAIL
- rename CHECKOUTPUT -> CHECKENTRY
- outpoint = SHA3(txid || entry_index || SHA3(output))
