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

        public static Key create(Context ctx) throws ChainException {
            Key key = ctx.request("mockhsm/create-key", null, Key.class);
            key.hsmUrl = buildMockHsmUrl(ctx.getUrl());
            return key;
        }

        public static class Items extends PagedItems<Key> {
            public Items getPage() throws ChainException {
                Items items = this.context.request("mockhsm/list-keys", this.query, Items.class);
                items.setContext(this.context);
                URL mockHsmUrl = buildMockHsmUrl(this.context.getUrl());
                for (Key k : items.list) {
                    k.hsmUrl = mockHsmUrl;
                }
                return items;
            }
        }

        public static Items list(Context ctx) throws ChainException {
            Items items = new Items();
            items.setContext(ctx);
            return items.getPage();
        }
    }

    private static URL buildMockHsmUrl(URL coreUrl) throws BadURLException {
        try {
            return new URL(coreUrl.toString() + "/mockhsm");
        } catch (MalformedURLException e) {
            throw new BadURLException(e.getMessage());
        }
    }
}
