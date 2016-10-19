import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class TransactionBasics {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    // snippet list-alice-transactions
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("inputs(account_alias=$1) AND outputs(account_alias=$1)")
      .addFilterParameter("alice")
      .execute(context);
    // endsnippet

    // snippet list-local-transactions
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(context);
    // endsnippet
  }
}
