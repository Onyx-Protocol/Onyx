package com.chain.analytics;

import com.google.gson.*;

import javax.sql.DataSource;
import java.io.Reader;
import java.lang.reflect.Type;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.*;

public class Config {
  private static final Serialization customColumnSerialization = new Serialization();
  private static final Gson gson =
      new GsonBuilder().registerTypeAdapter(CustomColumn.class, customColumnSerialization).create();

  final List<CustomColumn> transactionColumns;
  final List<CustomColumn> inputColumns;
  final List<CustomColumn> outputColumns;

  public Config() {
    transactionColumns = new ArrayList<>();
    inputColumns = new ArrayList<>();
    outputColumns = new ArrayList<>();
  }

  /**
   * reads a Chain Analytics configuration from JSON.
   *
   * @param  reader the JSON stream
   * @return        a Chain Analytics configuration
   */
  public static Config readFromJSON(final Reader reader)
      throws JsonIOException, JsonSyntaxException, InvalidConfigException {
    final Config config = gson.fromJson(reader, Config.class);
    if (config == null) {
      throw new InvalidConfigException("Unable to parse JSON config");
    }
    return config;
  }

  /**
   * CustomColumn is a user-configured column populated by extracting
   * data from json reference data.
   */
  public static class CustomColumn {
    final String name;
    final Schema.SQLType type;
    final JsonPath jsonPath;

    public CustomColumn(String name, Schema.SQLType type, JsonPath jsonPath) {
      this.name = name;
      this.type = type;
      this.jsonPath = jsonPath;
    }

    public boolean equals(final Object other) {
      if (other == null || !(other instanceof CustomColumn)) {
        return false;
      }
      final CustomColumn cc = (CustomColumn) other;
      return this.name.equals(cc.name)
          && this.type.toString().equals(cc.type.toString())
          && this.jsonPath.equals(cc.jsonPath);
    }

    public int hashCode() {
      int result = 21;
      result = result * 37 + name.hashCode();
      result = result * 37 + type.toString().hashCode();
      result = result * 37 + jsonPath.hashCode();
      return result;
    }
  }

  /**
   * Compares this config object to the provided one and returns
   * the set of removed and added columns.
   *
   * @param  target the desired configuration
   * @return        the set of columns to be added and removed to
   *                achieve target
   */
  public Migration diff(final Config target) {
    final Migration mig = new Migration();
    mig.removed.put("transaction_inputs", diffColumns(inputColumns, target.inputColumns));
    mig.removed.put("transaction_outputs", diffColumns(outputColumns, target.outputColumns));
    mig.removed.put("transactions", diffColumns(transactionColumns, target.transactionColumns));
    mig.added.put("transaction_inputs", diffColumns(target.inputColumns, inputColumns));
    mig.added.put("transaction_outputs", diffColumns(target.outputColumns, outputColumns));
    mig.added.put("transactions", diffColumns(target.transactionColumns, transactionColumns));
    return mig;
  }

  private List<CustomColumn> diffColumns(final List<CustomColumn> a, final List<CustomColumn> b) {
    final List<CustomColumn> l = new ArrayList<>(a);
    l.removeAll(b);
    return l;
  }

  public static class InvalidConfigException extends Exception {
    private static final long serialVersionUID = 201775336232807009L;

    public InvalidConfigException(String message) {
      super(message);
    }
  }

  public static class Migration {
    public Map<String, List<CustomColumn>> added;
    public Map<String, List<CustomColumn>> removed;

    public Migration() {
      added = new TreeMap<>();
      removed = new TreeMap<>();
    }
  }

  public static class Serialization
      implements JsonSerializer<CustomColumn>, JsonDeserializer<CustomColumn> {
    @Override
    public JsonElement serialize(
        CustomColumn src, Type typeOfSrc, JsonSerializationContext context) {
      JsonObject obj = new JsonObject();
      obj.add("name", new JsonPrimitive(src.name));
      obj.add("type", new JsonPrimitive(src.type.toString()));
      obj.add("path", new JsonPrimitive(src.jsonPath.toString()));
      return obj;
    }

    @Override
    public CustomColumn deserialize(
        JsonElement json, Type typeOfT, JsonDeserializationContext context)
        throws JsonParseException {
      final JsonObject jsonObject = json.getAsJsonObject();
      final String name = jsonObject.get("name").getAsString();
      final String typ = jsonObject.get("type").getAsString();
      final String path = jsonObject.get("path").getAsString();
      return new CustomColumn(name, OracleTypes.parse(typ), new JsonPath(path));
    }
  }
}
