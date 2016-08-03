package com.chain.signing;

import com.chain.api.TransactionTemplate;

import java.util.List;

public class HsmSigner {
    public List<KeyHandle> keys;
    HsmSigner(List<KeyHandle> keys) {
        this.keys = keys;
    }
    public void addKey(KeyHandle key) {
        this.keys.add(key);
    }
    public TransactionTemplate sign(List<TransactionTemplate> tmpls) { return null; }
}
