package com.chain.api;

import com.chain.exception.*;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.math.BigInteger;
import java.security.SecureRandom;
import java.text.SimpleDateFormat;
import java.time.OffsetDateTime;
import java.util.*;

/**
 * A single transaction on a Chain Core.
 */
public class Transaction {
  /**
   * Height of the block containing a transaction.
   */
  @SerializedName("block_height")
  public int blockHeight;

  /**
   * Unique identifier, or block hash, of the block containing a transaction.
   */
  @SerializedName("block_id")
  public String blockId;

  /**
   * Unique identifier, or transaction hash, of a transaction.
   */
  public String id;

  /**
   * List of specified inputs for a transaction.
   */
  public List<Input> inputs;

  /**
   * List of specified outputs for a transaction.
   */
  public List<Output> outputs;

  /**
   * Position of a transaction within the block.
   */
  public int position;

  /**
   * Time of transaction.
   */
  public Date timestamp;

  /**
   * User specified, unstructured data embedded within a transaction.
   */
  @SerializedName("reference_data")
  public Map<String, Object> referenceData;

  /**
   * Paged results of a transaction query.
   */
  public static class Items extends PagedItems<Transaction> {
    /**
     * Returns a new page of transactions based on the underlying query.
     * @return a page of transactions
     * @throws APIException This exception is raised if the api returns errors while processing the query.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items getPage() throws ChainException {
      Items items = this.context.request("list-transactions", this.query, Items.class);
      items.setContext(this.context);
      return items;
    }
  }

  /**
   * A builder class for transaction queries.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes a transaction query based on provided parameters.
     * @param ctx context object which makes server requests
     * @return a page of transactions
     * @throws APIException This exception is raised if the api returns errors while processing the query.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items execute(Context ctx) throws ChainException {
      Items items = new Items();
      items.setContext(ctx);
      items.setQuery(this.query);
      return items.getPage();
    }

    /**
     * Sets the earliest transaction timestamp to include in results
     * @param time start time in UTC format
     * @return updated QueryBuilder object
     */
    public QueryBuilder setStartTime(long time) {
      this.query.startTime = time;
      return this;
    }

    /**
     * Sets the latest transaction timestamp to include in results
     * @param time end time in UTC format
     * @return updated QueryBuilder object
     */
    public QueryBuilder setEndTime(long time) {
      this.query.endTime = time;
      return this;
    }
  }

  /**
   * A single input included in a transaction.
   */
  public static class Input {
    /**
     * The type of action being taken on an input.<br>
     * Possible actions are "issue", "spend_account", and "spend_account_unspent_output".
     */
    public String action;

    /**
     * The number of units of the asset being issued or spent.
     */
    public BigInteger amount;

    /**
     * The id of the asset being issued or spent.
     */
    @SerializedName("asset_id")
    public String assetId;

    /**
     * The id of the account transferring the asset (possibly null if the input is an issuance or an unspent output is specified).
     */
    @SerializedName("account_id")
    public String accountId;

    /**
     * The tags associated with the account (possibly null).
     */
    @SerializedName("account_tags")
    public Map<String, Object> accountTags;

    /**
     * The tags associated with the asset (possibly null).
     */
    @SerializedName("asset_tags")
    public Map<String, Object> assetTags;

    /**
     * Inputs to the control program used to verify the ability to take the specified action (possibly null).
     */
    @SerializedName("input_witness")
    public String[] inputWitness;

    /**
     * A program specifying a predicate for issuing an asset (possibly null if input is not an issuance).
     */
    @SerializedName("issuance_program")
    public String issuanceProgram;

    /**
     * User specified, unstructured data embedded within an input (possibly null).
     */
    @SerializedName("reference_data")
    public Map<String, Object> referenceData;
  }

  /**
   * A single output included in a transaction.
   */
  public static class Output {
    /**
     * The type of action being taken on the output.<br>
     * Possible actions are "control_account", "control_program", and "retire".
     */
    public String action;

    /**
     * The number of units of the asset being controlled.
     */
    public BigInteger amount;

    /**
     * The id of the asset being controlled.
     */
    @SerializedName("asset_id")
    public String assetId;

    /**
     * The control program which must be satisfied to transfer this output.
     */
    @SerializedName("control_program")
    public String controlProgram;

    /**
     * The output's position in a transaction's list of outputs
     */
    public int position;

    /**
     * The id of the account controlling this output (possibly null if a control program is specified).
     */
    @SerializedName("account_id")
    public String accountId;

    /**
     * The tags associated with the account controlling this output (possibly null if a control program is specified).
     */
    @SerializedName("account_tags")
    public Map<String, Object> accountTags;

    /**
     * The tags associated with the asset being controlled.
     */
    @SerializedName("asset_tags")
    public Map<String, Object> assetTags;

    /**
     * User specified, unstructured data embedded within an input (possibly null).
     */
    @SerializedName("reference_data")
    public Map<String, Object> referenceData;
  }

  /**
   * A built transaction that has not been submitted for block inclusion (returned from {@link Transaction#build(Context, List)}).
   */
  public static class Template {
    /**
     * A hex-encoded representation of a transaction template.
     */
    @SerializedName("raw_transaction")
    public String rawTransaction;

    /**
     * The list of signing instructions for inputs in the transaction.
     */
    @SerializedName("signing_instructions")
    public List<SigningInstruction> signing_instructions;

    /**
     * For core use only.
     */
    private Boolean local;

    /**
     * Set to true to make the transaction "final" when signing, preventing further changes.
     */
    @SerializedName("final")
    public Boolean isFinal;

    /**
     * The Chain error code.
     */
    public String code;

    /**
     * The Chain error message.
     */
    public String message;

    /**
     * Additional details about the error.
     */
    public String detail;

    /**
     * A single signing instruction included in a transaction template.
     */
    public static class SigningInstruction {
      /**
       * The id of the asset being issued or spent.
       */
      @SerializedName("asset_id")
      public String assetID;

      /**
       * The number of units of the asset being issued or spent.
       */
      public BigInteger amount;

      /**
       * The input's position in a transaction's list of inputs.
       */
      public int position;

      /**
       * A list of components used to coordinate the signing of an input.
       */
      @SerializedName("witness_components")
      public WitnessComponent[] witnessComponents;
    }

    /**
     * A single witness component, holding information that will become the input witness.
     */
    public static class WitnessComponent {
      /**
       * The type of witness component.<br>
       * Possible types are "data" and "signature".
       */
      public String type;

      /**
       * Data to be included in the input witness (null unless type is "data").
       */
      public String data;

      /**
       * The number of signatures required for an input (null unless type is "signature").
       */
      public int quorum;

      /**
       * The list of keys to sign with (null unless type is "signature").
       */
      public KeyID[] keys;

      /**
       * The program whose hash is signed. If empty, it is
       * inferred during signing from aspects of the
       * transaction.
       */
      public String program;

      /**
       * The list of signatures made with the specified keys (null unless type is "signature").
       */
      public String[] signatures;
    }

    /**
     * A class representing a derived signing key.
     */
    public static class KeyID {
      /**
       * The extended public key associated with the private key used to sign.
       */
      public String xpub;

      /**
       * The derivation path of the extended public key.
       */
      @SerializedName("derivation_path")
      public ArrayList<Integer> derivationPath;
    }
  }

  /**
   * A single response from a call to {@link Transaction#submit(Context, List)}
   */
  public static class SubmitResponse {
    /**
     * The transaction id.
     */
    public String id;

    /**
     * The Chain error code.
     */
    public String code;

    /**
     * The Chain error message.
     */
    public String message;

    /**
     * Additional details about the error.
     */
    public String detail;
  }

  /**
   * Builds a transaction template.
   * @param ctx context object which makes server requests
   * @param builders list of transaction builders
   * @return a list of transaction templates
   * @throws APIException This exception is raised if the api returns errors while building transaction templates.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static List<Template> build(Context ctx, List<Transaction.Builder> builders)
      throws ChainException {
    SecureRandom random = new SecureRandom();
    SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd'T'hh:mm:ss'Z'");

    for (Builder builder : builders) {
      for (Action action : builder.actions) {
        if (action.get("type") == "issue") {
          StringBuilder sb = new StringBuilder();
          while (sb.length() < 4) {
            sb.append(Integer.toHexString(random.nextInt()));
          }
          action.setParameter("nonce", sb.toString());
          action.setParameter("min_time", formatter.format(new Date()));
        }
      }
    }
    Type type = new TypeToken<ArrayList<Template>>() {}.getType();
    return ctx.request("build-transaction", builders, type);
  }

  /**
   * Submits signed transaction templates for inclusion into a block.
   * @param ctx context object which makes server requests
   * @param templates list of transaction templates
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static List<SubmitResponse> submit(Context ctx, List<Template> templates)
      throws ChainException {
    Type type = new TypeToken<ArrayList<SubmitResponse>>() {}.getType();

    HashMap<String, Object> requestBody = new HashMap<>();
    requestBody.put("transactions", templates);

    return ctx.request("submit-transaction", requestBody, type);
  }

  /**
   * An action that can be taken within a transaction.
   */
  public static class Action extends HashMap<String, Object> {
    /**
     * Default constructor initializes list and sets the client token.
     */
    public Action() {
      // Several action types require client_token as an idempotency key.
      // It's safest to include a default value for this param.
      this.put("client_token", UUID.randomUUID().toString());
    }

    /**
     * Sets a k,v parameter pair.
     * @param key the key on the parameter object
     * @param value the corresponding value
     * @return updated action object
     */
    public Action setParameter(String key, Object value) {
      this.put(key, value);
      return this;
    }
  }

  /**
   * A builder class for transaction templates.
   */
  public static class Builder {
    /**
     * List of actions in a transaction.
     */
    private List<Action> actions;

    /**
     * User specified, unstructured data embedded at the top level of the transaction.
     */
    @SerializedName("reference_data")
    private Map<String, Object> referenceData;

    /**
     * A time duration in milliseconds. If the transaction is not fully
     * signed and submitted within this time, it will be rejected by the
     * blockchain. Additionally, any outputs reserved when building this
     * transaction will remain reserved for this duration.
     */
    private long ttl;

    /**
     * Builds a single transaction template.
     * @param ctx context object which makes requests to the server
     * @return a transaction template
     * @throws APIException This exception is raised if the api returns errors while building the transaction.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Template build(Context ctx) throws ChainException {
      List<Template> tmpls = Transaction.build(ctx, Arrays.asList(this));
      Template response = tmpls.get(0);
      if (response.rawTransaction == null) {
        throw new APIException(response.code, response.message, response.detail, null);
      }
      return response;
    }

    /**
     * Default constructor initializes actions list.
     */
    public Builder() {
      this.actions = new ArrayList<>();
    }

    /**
     * Adds an action to a transaction builder.
     * @param action action to add
     * @return updated builder object
     */
    public Builder addAction(Action action) {
      this.actions.add(action);
      return this;
    }

    /**
     * Sets a transaction's reference data.
     * @param referenceData info to embed into a transaction.
     * @return
     */
    public Builder setReferenceData(Map<String, Object> referenceData) {
      this.referenceData = referenceData;
      return this;
    }

    /**
     * Sets a transaction's time-to-live, which indicates how long outputs
     * will be reserved for, and how long the transaction will remain valid.
     * Passing zero will use the default TTL, which is 300000ms (5 minutes).
     * @param ms the duration of the TTL, in milliseconds.
     * @return updated builder object
     */
    public Builder setTtl(long ms) {
      this.ttl = ms;
      return this;
    }

    /**
     * Adds an issuance action to a transaction, using an id to specify the asset.
     * @param assetId id of the asset being issued
     * @param amount number of units of the asset to issue
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder issueById(String assetId, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "issue")
              .setParameter("asset_id", assetId)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds an issuance action to a transaction, using an alias to specify the asset.
     * @param assetAlias alias of the asset being issued
     * @param amount number of units of the asset to issue
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     * @return
     */
    public Builder issueByAlias(
        String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "issue")
              .setParameter("asset_alias", assetAlias)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a control (by account) action to a transaction, using an id to specify the asset and account.
     * @param accountId id of the account controlling the asset
     * @param assetId id of the asset being controlled
     * @param amount number of units of the asset being controlled
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder controlWithAccountById(
        String accountId, String assetId, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_account")
              .setParameter("account_id", accountId)
              .setParameter("asset_id", assetId)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a control (by account) action to a transaction, using an alias to specify the asset and account.
     * @param accountAlias alias of the account controlling the asset
     * @param assetAlias alias of the asset being controlled
     * @param amount number of units of the asset being controlled
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder controlWithAccountByAlias(
        String accountAlias,
        String assetAlias,
        BigInteger amount,
        Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_account")
              .setParameter("account_alias", accountAlias)
              .setParameter("asset_alias", assetAlias)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a control (by control program) action to a transaction, using an id to specify the asset.
     * @param program control program which will control the asset
     * @param assetId id of the asset being controlled
     * @param amount number of units of the asset being controlled
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder controlWithProgramById(
        ControlProgram program,
        String assetId,
        BigInteger amount,
        Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_program")
              .setParameter("control_program", program.program)
              .setParameter("asset_id", assetId)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a control (by control program) action to a transaction, using an alias to specify the asset.
     * @param program control program which will control the asset
     * @param assetAlias alias of the asset being controlled
     * @param amount number of units of the asset being controlled
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder controlWithProgramByAlias(
        ControlProgram program,
        String assetAlias,
        BigInteger amount,
        Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_program")
              .setParameter("control_program", program.program)
              .setParameter("asset_alias", assetAlias)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a spend (by account) action to a transaction, using an id to specify the account and asset.
     * @param accountId id of the account spending the asset
     * @param assetId id of the asset being spent
     * @param amount number of units of the asset being spent
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder spendFromAccountById(
        String accountId, String assetId, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "spend_account")
              .setParameter("account_id", accountId)
              .setParameter("asset_id", assetId)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a spend (by account) action to a transaction, using an alias to specify the account and asset.
     * @param accountAlias alias of the account spending the asset
     * @param assetAlias alias of the asset being spent
     * @param amount number of units of the asset being spent
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder spendFromAccountByAlias(
        String accountAlias,
        String assetAlias,
        BigInteger amount,
        Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "spend_account")
              .setParameter("account_alias", accountAlias)
              .setParameter("asset_alias", assetAlias)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a spend (by unspent output) action to a transaction.
     * @param unspentOutput unspent output to spend
     * @param referenceData reference data to embed into action (possibly null)
     * @return
     */
    public Builder spendUnspentOutput(
        UnspentOutput unspentOutput, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "spend_account_unspent_output")
              .setParameter("transaction_id", unspentOutput.transactionId)
              .setParameter("position", unspentOutput.position)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a list of spend (by unspent output) actions to a transaction.
     * @param uos list of unspent outputs to spend
     * @param referenceData reference data to embed into each action (possibly null)
     * @return updated builder object
     */
    public Builder spendUnspentOutputs(List<UnspentOutput> uos, Map<String, Object> referenceData) {
      for (UnspentOutput uo : uos) {
        this.spendUnspentOutput(uo, referenceData);
      }

      return this;
    }

    /**
     * Adds a retire action to a transaction, using an id to specify the asset.
     * @param assetId id of the asset to retire
     * @param amount number of units of the asset to retire
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder retireById(
        String assetId, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_program")
              .setParameter("control_program", ControlProgram.retireProgram())
              .setParameter("asset_id", assetId)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }

    /**
     * Adds a retire action to a transaction, using an alias to specify the asset.
     * @param assetAlias id of the asset to retire
     * @param amount number of units of the asset to retire
     * @param referenceData reference data to embed into the action (possibly null)
     * @return updated builder object
     */
    public Builder retireByAlias(
        String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
      Action action =
          new Action()
              .setParameter("type", "control_program")
              .setParameter("control_program", ControlProgram.retireProgram())
              .setParameter("asset_alias", assetAlias)
              .setParameter("amount", amount)
              .setParameter("reference_data", referenceData);

      return this.addAction(action);
    }
  }
}
