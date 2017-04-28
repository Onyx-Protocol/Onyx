package com.chain.integration;

import com.chain.api.*;
import com.chain.http.*;

import org.junit.Test;
import static org.junit.Assert.*;

public class CoreConfigTest {
  @Test
  public void run() throws Exception {
    Client c = new Client();

    CoreConfig.Info info = CoreConfig.getInfo(c);
    assertTrue(info.isConfigured);

    CoreConfig.resetEverything(c);
    info = CoreConfig.getInfo(c);
    assertFalse(info.isConfigured);

    new CoreConfig.Builder().setIsGenerator(true).configure(c);
    info = CoreConfig.getInfo(c);
    assertTrue(info.isConfigured);

    assertTrue(info.health.errors.size() == 0);
  }
}
