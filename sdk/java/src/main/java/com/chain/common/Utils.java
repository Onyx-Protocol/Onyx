package com.chain.common;

import com.google.gson.*;

public class Utils {
  public static String rfc3339DateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSXXX";
  public static final Gson serializer = new GsonBuilder().setDateFormat(rfc3339DateFormat).create();
}
