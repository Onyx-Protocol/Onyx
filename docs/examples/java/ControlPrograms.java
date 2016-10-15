import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class ControlPrograms {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    // snippet create-control-program
    ControlProgram aliceProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(context);
    // endsnippet

    // snippet build-transaction
    Transaction.Template transaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.program)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);
    // endsnippet

    // snippet retire
    Transaction.Template retirement = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("Alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(retirement));
    // endsnippet
  }
}
