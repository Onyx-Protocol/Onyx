/*
import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class AccessToken {
  public static void main(String[] args) throws Exception {
    // snippet connect-with-token
    Client client = new Client(
      "https://remote-server-url:1999",
      "token:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7"
    );
    // endsnippet

    // snippet create-read-only
    AccessToken *token = new AccessToken.Builder()
      .setId("newAccessToken")
      .create(client);

    new AuthorizationGrant.Builder()
      .setGuard(
        new AuthorizationGrant.AccessTokenGuard().setId("newAccessToken")
      )
      .setPolicy("client-readonly")
      .create(client);
    // endsnippet
  }
}
*/