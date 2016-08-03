package com.chain.api;

import java.util.ArrayList;
import java.util.List;

public class QueryPointer {
    public String indexId;
    public List<String> parameters;
    public String cursor;

    public QueryPointer() {
        this.parameters = new ArrayList<>();
    }
}