package analytics;

import java.util.ArrayList;
import java.util.List;

public class Config {
  List<CustomColumn> transactionColumns;
  List<CustomColumn> inputColumns;
  List<CustomColumn> outputColumns;

  public Config() {
    transactionColumns = new ArrayList<>();
    inputColumns = new ArrayList<>();
    outputColumns = new ArrayList<>();
  }

  public static class CustomColumn {
    String name;
    Schema.SQLType type;
    JsonPath jsonPath;

    public CustomColumn(String name, Schema.SQLType type, JsonPath jsonPath) {
      this.name = name;
      this.type = type;
      this.jsonPath = jsonPath;
    }
  }
}
