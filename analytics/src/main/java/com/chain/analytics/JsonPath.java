package com.chain.analytics;

import com.chain.api.Transaction;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;

/**
 * A JsonPath specifies a path to a value within a JSON object.
 *
 * TODO(jackson): Do we need to support indexing into arrays?
 */
class JsonPath {
  private final List<String> mPath;

  public JsonPath(final List<String> path) {
    mPath = Collections.unmodifiableList(path);
  }

  public JsonPath(final String serializedPath) {
    mPath = Collections.unmodifiableList(Arrays.asList(serializedPath.split("[.]{1}")));
  }

  public String toString() {
    return String.join(".", mPath);
  }

  public boolean equals(final Object other) {
    if (other == null || !(other instanceof JsonPath)) {
      return false;
    }
    return mPath.equals(((JsonPath) other).mPath);
  }

  public int hashCode() {
    return mPath.hashCode();
  }

  public Object extract(final Transaction tx) {
    if (mPath.isEmpty()) {
      return tx;
    }

    // The first component in the path is a top-level transaction
    // field. All the subsequent fields index into unstructured json.
    String field = mPath.get(0);

    Map<String, Object> deserializedJson = null;
    switch (field.toLowerCase()) {
      case "reference_data":
        deserializedJson = tx.referenceData;
        break;
    }
    return extract(deserializedJson, mPath.subList(1, mPath.size()));
  }

  public Object extract(final Transaction.Input input) {
    if (mPath.isEmpty()) {
      return input;
    }

    // The first component in the path is a top-level input
    // field. All the subsequent fields index into unstructured json.
    String field = mPath.get(0);

    Map<String, Object> deserializedJson = null;
    switch (field.toLowerCase()) {
      case "account_tags":
        deserializedJson = input.accountTags;
        break;
      case "asset_definition":
        deserializedJson = input.assetDefinition;
        break;
      case "asset_tags":
        deserializedJson = input.assetTags;
        break;
      case "reference_data":
        deserializedJson = input.referenceData;
        break;
    }
    return extract(deserializedJson, mPath.subList(1, mPath.size()));
  }

  public Object extract(final Transaction.Output output) {
    if (mPath.isEmpty()) {
      return output;
    }

    // The first component in the path is a top-level output
    // field. All the subsequent fields index into unstructured json.
    String field = mPath.get(0);

    Map<String, Object> deserializedJson = null;
    switch (field.toLowerCase()) {
      case "account_tags":
        deserializedJson = output.accountTags;
        break;
      case "asset_definition":
        deserializedJson = output.assetDefinition;
        break;
      case "asset_tags":
        deserializedJson = output.assetTags;
        break;
      case "reference_data":
        deserializedJson = output.referenceData;
        break;
    }
    return extract(deserializedJson, mPath.subList(1, mPath.size()));
  }

  @SuppressWarnings("unchecked")
  private Object extract(final Map<String, Object> deserializedJson, final List<String> path) {
    Map<String, Object> jsonObject = deserializedJson;
    if (path.isEmpty()) {
      return jsonObject;
    }

    try {
      // Follow all of the path elements except for the last.
      for (final String key : path.subList(0, path.size() - 1)) {
        if (jsonObject == null) {
          return null;
        }

        final Object v = jsonObject.get(key);
        if (v == null) {
          return null;
        }
        jsonObject = (Map<String, Object>) v;
      }
    } catch (ClassCastException ex) {
      return null;
    }

    // Follow the final path element.
    return jsonObject.get(path.get(path.size() - 1));
  }
}
