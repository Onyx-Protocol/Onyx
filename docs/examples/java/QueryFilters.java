import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class QueryFilters {
  public static void main(String[] args) throws Exception {
    Client client = new Client();

    // snippet list-alice-transactions
    Transaction.Items aliceTransaction = new Transaction.QueryBuilder()
      .setFilter("inputs(account_alias=$1) AND outputs(account_alias=$1)")
      .addFilterParameter("alice")
      .execute(client);
    // endsnippet

    // snippet list-local-transactions
    Transaction.Items localTransactions = new Transaction.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(client);
    // endsnippet
  }
}
