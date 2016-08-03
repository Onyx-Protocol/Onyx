package com.chain.signing;

import com.chain.api.BasePage;
import com.chain.api.BaseQuery;
import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.net.URL;

/**
 * A key handle object consisting of a public key and authenticated
 * URL. Used to sign objects with the HSM specified in the URL.
 */
public class KeyHandle {
    private String xpub;
    private URL hsmUrl;

    public KeyHandle(String xpub, URL hsmUrl) {
        this.xpub = xpub;
        this.hsmUrl = hsmUrl;
    }

    public String getXPub() {
        return xpub;
    }
    public void setXPub(String xpub) { this.xpub = xpub; }
    public URL getHsmUrl() { return hsmUrl; }
    public void setHsmUrl(URL hsmUrl) { this.hsmUrl = hsmUrl; }


    public static class Page extends BasePage<KeyHandle> {
        public Page next(Context ctx)
        throws ChainException {
            Page page = ctx.request("list-keys", this.queryPointer, Page.class);
            for (KeyHandle key : page.items) {
                key.setHsmUrl(ctx.getUrl());
            }
            return page;
        }
    }

    public static class Query extends BaseQuery<Page> {
        public Page search(Context ctx)
        throws ChainException {
            Page page = ctx.request("list-keys", this.queryPointer, Page.class);
            for (KeyHandle key : page.items) {
                key.setHsmUrl(ctx.getUrl());
            }
            return page;
        }
    }

    public static class Builder {
        /**
         * Generate a new KeyHandle with the HSM specified.
         *
         * @param ctx A context objec pointing to an HSM
         * @return New key handle ready for use
         */
        public KeyHandle create(Context ctx)
        throws Exception {
            KeyHandle key = ctx.request("create-key", this, KeyHandle.class);
            key.setHsmUrl(ctx.getUrl());
            return key;
        }
    }
}
