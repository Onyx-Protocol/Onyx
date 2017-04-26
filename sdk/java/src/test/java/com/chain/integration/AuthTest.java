package com.chain.integration;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.TestUtils;

import java.util.*;

import static org.junit.Assert.*;
import org.junit.Test;

public class AuthTest {
  @Test
  public void accessTokens() throws Exception {
    Client client = TestUtils.generateClient();

    String tokenID = "test" + UUID.randomUUID();

    AccessToken token = new AccessToken.Builder().setId(tokenID).create(client);

    assertEquals(token.id, tokenID);
    assertTrue(token.token.length() > 0);

    boolean found = false;
    AccessToken.Items items = new AccessToken.QueryBuilder().execute(client);
    while (items.hasNext()) {
      AccessToken t = items.next();
      if (t.id.equals(tokenID)) {
        found = true;
        break;
      }
    }
    assertTrue(found);
  }

  @Test
  public void authorizationGrants() throws Exception {
    Client client = TestUtils.generateClient();

    String tokenID = "test" + UUID.randomUUID();

    new AccessToken.Builder().setId(tokenID).create(client);

    // Grant with access token guard

    new AuthorizationGrant.Builder()
        .setGuard(new AuthorizationGrant.AccessTokenGuard().setId(tokenID))
        .setPolicy("monitoring")
        .create(client);

    boolean found = false;
    List<AuthorizationGrant> list = AuthorizationGrant.listAll(client);
    for (AuthorizationGrant g : list) {
      if (!g.guardType.equals("access_token")) continue;
      if (!g.policy.equals("monitoring")) continue;

      Object id = g.guardData.get("id");
      if (!(id instanceof String)) continue;
      if (!((String) id).equals(tokenID)) continue;

      found = true;
      break;
    }
    assertTrue(found);

    // Revocation

    new AuthorizationGrant.DeletionBuilder()
        .setGuard(new AuthorizationGrant.AccessTokenGuard().setId(tokenID))
        .setPolicy("monitoring")
        .delete(client);

    found = false;
    list = AuthorizationGrant.listAll(client);
    for (AuthorizationGrant g : list) {
      if (!g.guardType.equals("access_token")) continue;
      if (!g.policy.equals("monitoring")) continue;

      Object id = g.guardData.get("id");
      if (!(id instanceof String)) continue;
      if (!((String) id).equals(tokenID)) continue;

      found = true;
      break;
    }
    assertFalse(found);

    // Grant with X509 guard

    new AuthorizationGrant.Builder()
        .setGuard(
            new AuthorizationGrant.X509Guard()
                .setCommonName("test-cn")
                .addOrganizationalUnit("test-ou"))
        .setPolicy("monitoring")
        .create(client);

    found = false;
    list = AuthorizationGrant.listAll(client);
    for (AuthorizationGrant g : list) {
      if (!g.guardType.equals("x509")) continue;
      if (!g.policy.equals("monitoring")) continue;

      Object subject = g.guardData.get("subject");
      if (!(subject instanceof Map)) continue;
      Map<String, Object> subjectMap = (Map<String, Object>) subject;

      Object cn = subjectMap.get("CN");
      if (!(cn instanceof String)) continue;
      if (!((String) cn).equals("test-cn")) continue;

      Object ou = subjectMap.get("OU");
      if (!(ou instanceof List)) continue;
      List<String> ouValues = (List<String>) ou;
      if (ouValues.size() != 1) continue;
      if (!ouValues.get(0).equals("test-ou")) continue;

      found = true;
      break;
    }
    assertTrue(found);
  }
}
