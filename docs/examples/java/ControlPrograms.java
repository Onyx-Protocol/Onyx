import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class ControlPrograms {
  public static void main(String[] args) throws Exception {
    Context context = new Context();
    setup(context);

    // snippet create-control-program
    ControlProgram aliceProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("alice")
      .create(context);
    // endsnippet

    // snippet build-transaction
    Transaction.Template paymentToProgram = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(aliceProgram.program)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(paymentToProgram));
    // endsnippet

    // snippet retire
    Transaction.Template retirement = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(retirement));
    // endsnippet
  }

  public static void setup(Context context) throws Exception {
    MockHsm.Key key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    Transaction.submit(context, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(context)
    ));
  }
}
