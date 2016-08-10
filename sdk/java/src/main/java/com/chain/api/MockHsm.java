package com.chain.api;

import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.net.MalformedURLException;
import java.net.URL;

public class MockHsm {
    public static class Key {
        public String xpub;
        public URL hsmUrl;

        public static class Page extends BasePage<Key> {
            public Page next(Context ctx)
            throws ChainException {
                Key.Page page = ctx.request("mockhsm/list-keys", this.queryPointer, Page.class);
                URL mockHsmUrl = buildMockHsmUrl(ctx.getUrl());
                if (page.items != null) {
                    for (Key k : page.items) {
                        k.hsmUrl = mockHsmUrl;
                    }
                }
                return page;
            }
        }

        public static Key create(Context ctx)
        throws ChainException {
            Key key = ctx.request("mockhsm/create-key", null, Key.class);
            key.hsmUrl = buildMockHsmUrl(ctx.getUrl());
            return key;
        }

        public static Key.Page list(Context ctx)
        throws ChainException {
            Key.Page page = ctx.request("mockhsm/list-keys", null, Page.class);
            URL mockHsmUrl = buildMockHsmUrl(ctx.getUrl());
            if (page.items != null) {
                for (Key k : page.items) {
                    k.hsmUrl = mockHsmUrl;
                }
            }
            return page;
        }
    }

    private static URL buildMockHsmUrl(URL coreUrl)
    throws BadURLException {
        try {
            return new URL(coreUrl.toString() + "/mockhsm");
        } catch (MalformedURLException e) {
            throw new BadURLException(e.getMessage());
        }
    }
}
