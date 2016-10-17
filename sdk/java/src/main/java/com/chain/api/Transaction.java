package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;

import java.util.*;

/**
 * A single transaction on a Chain Core.
 */
public class Transaction {
  /**
   * Unique identifier, or transaction hash, of a transaction.
   */
  public String id;

  /**
   * Time of transaction.
   */
  public Date timestamp;

  /**
   * Unique identifier, or block hash, of the block containing a transaction.
   */
  @SerializedName("block_id")
  public String blockId;

  /**
   * Height of the block containing a transaction.
   */
  @SerializedName("block_height")
  public int blockHeight;

  /**
   * Position of a transaction within the block.
   */
  public int position;

  /**
   * User specified, unstructured data embedded within a transaction.
   */
  @SerializedName("reference_data")
  public Map<String, Object> referenceData;

  /**
   * A flag indicating one or more inputs or outputs are local.
   * Possible values are "yes" or "no".
   */
  @SerializedName("is_local")
  public String isLocal;

  /**
   * List of specified inputs for a transaction.
   */
  public List<Input> inputs;

  /**
   * List of specified outputs for a transaction.
   */
  public List<Output> outputs;

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
      Items items = this.context.request("list-transactions", this.next, Items.class);
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
      items.setNext(this.next);
      return items.getPage();
    }

    /**
     * Sets the earliest transaction timestamp to include in results
     * @param time start time in UTC format
     * @return updated QueryBuilder object
     */
    public QueryBuilder setStartTime(long time) {
      this.next.startTime = time;
      return this;
    }

    /**
     * Sets the latest transaction timestamp to include in results
     * @param time end time in UTC format
     * @return updated QueryBuilder object
     */
    public QueryBuilder setEndTime(long time) {
      this.next.endTime = time;
      return this;
    }

    /**
     * Sets the ascending_with_long_poll flag on this query to facilitate
     * notifications.
     * @return updated QueryBuilder object
     */
    public QueryBuilder setAscendingWithLongPoll() {
      this.next.ascendingWithLongPoll = true;
      return this;
    }

    /**
     * Sets a timeout on this query.
     * @param timeoutMS timeout in milliseconds
     * @return updated QueryBuilder object
     */
    public QueryBuilder setTimeout(long timeoutMS) {
      this.next.timeout = timeoutMS;
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
    public String type;

    /**
     * The id of the asset being issued or spent.
     */
    @SerializedName("asset_id")
    public String assetId;

    /**
     * The alias of the asset being issued or spent (possibly null).
     */
    @SerializedName("asset_alias")
    public String assetAlias;

    /**
     * The definition of the asset being issued or spent (possibly null).
     */
    @SerializedName("asset_definition")
    public Map<String, Object> assetDefinition;

    /**
     * The tags of the asset being issued or spent (possibly null).
     */
    @SerializedName("asset_tags")
    public Map<String, Object> assetTags;

    /**
     * A flag indicating whether the asset being issued or spent is local.
     * Possible values are "yes" or "no".
     */
    @SerializedName("asset_is_local")
    public String assetIsLocal;

    /**
     * The number of units of the asset being issued or spent.
     */
    public long amount;

    /**
     * The id of the account transferring the asset (possibly null if the input is an issuance or an unspent output is specified).
     */
    @SerializedName("account_id")
    public String accountId;

    /**
     * The output consumed by this input. Null if the input is an issuance.
     */
    @SerializedName("spent_output")
    public OutputPointer spentOutput;

    /**
     * The alias of the account transferring the asset (possibly null if the input is an issuance or an unspent output is specified).
     */
    @SerializedName("account_alias")
    public String accountAlias;

    /**
     * The tags associated with the account (possibly null).
     */
    @SerializedName("account_tags")
    public Map<String, Object> accountTags;

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

    /**
     * A flag indicating if the input is local.
     * Possible values are "yes" or "no".
     */
    @SerializedName("is_local")
    public String isLocal;
  }

  /**
   * A single output included in a transaction.
   */
  public static class Output {
    /**
     * The type of action being taken on the output.<br>
     * Possible actions are "control_account", "control_program", and "retire".
     */
    public String type;

    /**
     * The purpose of the output.<br>
     * Possible purposes are "receive" and "change". Only populated if the
     * output's control program was generated locally.
     */
    public String purpose;

    /**
     * The output's position in a transaction's list of outputs.
     */
    public int position;

    /**
     * The id of the asset being controlled.
     */
    @SerializedName("asset_id")
    public String assetId;

    /**
     * The alias of the asset being controlled.
     */
    @SerializedName("asset_alias")
    public String assetAlias;

    /**
     * The definition of the asset being controlled (possibly null).
     */
    @SerializedName("asset_definition")
    public Map<String, Object> assetDefinition;

    /**
     * The tags of the asset being controlled (possibly null).
     */
    @SerializedName("asset_tags")
    public Map<String, Object> assetTags;

    /**
     * A flag indicating whether the asset being controlled is local.
     * Possible values are "yes" or "no".
     */
    @SerializedName("asset_is_local")
    public String assetIsLocal;

    /**
     * The number of units of the asset being controlled.
     */
    public long amount;

    /**
     * The id of the account controlling this output (possibly null if a control program is specified).
     */
    @SerializedName("account_id")
    public String accountId;

    /**
     * The alias of the account controlling this output (possibly null if a control program is specified).
     */
    @SerializedName("account_alias")
    public String accountAlias;

    /**
     * The tags associated with the account controlling this output (possibly null if a control program is specified).
     */
    @SerializedName("account_tags")
    public Map<String, Object> accountTags;

    /**
     * The control program which must be satisfied to transfer this output.
     */
    @SerializedName("control_program")
    public String controlProgram;

    /**
     * User specified, unstructured data embedded within an input (possibly null).
     */
    @SerializedName("reference_data")
    public Map<String, Object> referenceData;

    /**
     * A flag indicating if the output is local.
     * Possible values are "yes" or "no".
     */
    @SerializedName("is_local")
    public String isLocal;
  }

  /**
   * An OutputPointer consists of a transaction ID and an output position, and
   * uniquely identifies an output on the blockchain.
   */
  public static class OutputPointer {
    @SerializedName("transaction_id")
    public String transactionId;

    public int position;
  }

  /**
   * A built transaction that has not been submitted for block inclusion (returned from {@link Transaction#buildBatch(Context, List)}).
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
    public List<SigningInstruction> signingInstructions;

    /**
     * For core use only.
     */
    private boolean local;

    /**
     * False (the default) makes the transaction "final" when signing,
     * preventing further changes - the signature program commits to
     * the transaction's signature hash.  True makes the transaction
     * extensible, committing only to the elements in the transaction
     * so far, permitting the addition of new elements.
     */
    @SerializedName("allow_additional_actions")
    public boolean allowAdditionalActions;

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
      public long amount;

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
      public String[] derivationPath;
    }
  }

  /**
   * A single response from a call to {@link Transaction#submitBatch(Context, List)}
   */
  public static class SubmitResponse {
    /**
     * The transaction id.
     */
    public String id;
  }

  /**
   * Builds a batch of transaction templates.
   * @param ctx context object which makes server requests
   * @param builders list of transaction builders
   * @return a list of transaction templates
   * @throws APIException This exception is raised if the api returns errors while building transaction templates.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<Template> buildBatch(Context ctx, List<Transaction.Builder> builders)
      throws ChainException {
    return ctx.batchRequest("build-transaction", builders, Template.class);
  }

  /**
   * Submits a batch of signed transaction templates for inclusion into a block.
   * @param ctx context object which makes server requests
   * @param templates list of transaction templates
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<SubmitResponse> submitBatch(Context ctx, List<Template> templates)
      throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", templates);
    return ctx.batchRequest("submit-transaction", body, SubmitResponse.class);
  }

  /**
   * Submits signed transaction template for inclusion into a block.
   * @param ctx context object which makes server requests
   * @param template transaction template
   * @return submit responses
   * @throws APIException This exception is raised if the api returns errors while submitting a transaction.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static SubmitResponse submit(Context ctx, Template template) throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", Arrays.asList(template));
    return ctx.singletonBatchRequest("submit-transaction", body, SubmitResponse.class);
  }

  /**
   * Base class representing actions that can be taken within a transaction.
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
     * Adds a k,v pair to the action's reference data object.<br>
     * Since most/all current action types use the reference_data parameter, we provide this method in the base class to avoid repetition.
     * @param key key of the reference data field
     * @param value value of reference data field
     * @return updated action object
     */
    public Action addReferenceDataField(String key, Object value) {
      Map<String, Object> referenceData = (HashMap<String, Object>) this.get("reference_data");
      if (referenceData == null) {
        referenceData = new HashMap<>();
        this.put("reference_data", referenceData);
      }
      referenceData.put(key, value);
      return this;
    }

    /**
     * Specifies the reference data to associate with the action
     * Since most/all current action types use the reference_data parameter, we provide this method in the base class to avoid repetition.
     * @param referenceData reference data to embed into the action
     * @return updated action object
     */
    public Action setReferenceData(Map<String, Object> referenceData) {
      this.put("reference_data", referenceData);
      return this;
    }

    /**
     * Represents an issuance action.
     */
    public static class Issue extends Action {
      /**
       * Default constructor defines the action type as "issue"
       */
      public Issue() {
        this.put("type", "issue");
      }

      /**
       * Specifies the asset to be issued using its alias
       * @param alias alias of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be issued using its id
       * @param id id of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be issued
       * @param amount number of units of the asset to be issued
       * @return updated action object
       */
      public Issue setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }

      /**
       * Specifies the time to live for this action.
       * @param ttlMS the ttl, in milliseconds
       * @return updated action object
       */
      public Issue setTTL(long ttlMS) {
        this.put("ttl", ttlMS);
        return this;
      }
    }

    /**
     * Represents a spend action taken on a particular unspent output.
     */
    public static class SpendAccountUnspentOutput extends Action {
      /**
       * Default constructor defines the action type as "spend_account_unspent_output"
       */
      public SpendAccountUnspentOutput() {
        this.put("type", "spend_account_unspent_output");
      }

      /**
       * Specifies the unspent output to be spent
       * @param unspentOutput unspent output to be spent
       * @return updated action object
       */
      public SpendAccountUnspentOutput setUnspentOutput(UnspentOutput unspentOutput) {
        setTransactionId(unspentOutput.transactionId);
        setPosition(unspentOutput.position);
        return this;
      }

      public SpendAccountUnspentOutput setTransactionId(String id) {
        this.put("transaction_id", id);
        return this;
      }

      public SpendAccountUnspentOutput setPosition(int pos) {
        this.put("position", pos);
        return this;
      }

      /**
       * Specifies the time to live for this action.
       * @param ttlMS the ttl, in milliseconds
       * @return updated action object
       */
      public SpendAccountUnspentOutput setTTL(long ttlMS) {
        this.put("ttl", ttlMS);
        return this;
      }
    }

    /**
     * Represents a spend action taken on a particular account.
     */
    public static class SpendFromAccount extends Action {
      /**
       * Default constructor defines the action type as "spend_account"
       */
      public SpendFromAccount() {
        this.put("type", "spend_account");
      }

      /**
       * Specifies the spending account using its alias.<br>
       * <strong>Must</strong> be used with {@link SpendFromAccount#setAssetAlias(String)}
       * @param alias alias of the spending account
       * @return updated action object
       */
      public SpendFromAccount setAccountAlias(String alias) {
        this.put("account_alias", alias);
        return this;
      }

      /**
       * Specifies the spending account using its id.<br>
       * <strong>Must</strong> be used with {@link SpendFromAccount#setAssetId(String)}
       * @param id id of the spending account
       * @return updated action object
       */
      public SpendFromAccount setAccountId(String id) {
        this.put("account_id", id);
        return this;
      }

      /**
       * Specifies the asset to be spent using its alias
       * @param alias alias of the asset to be spent
       * <strong>Must</strong> be used with {@link SpendFromAccount#setAccountAlias(String)}}
       * @return updated action object
       */
      public SpendFromAccount setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be spent using its id
       * @param id id of the asset to be spent
       * <strong>Must</strong> be used with {@link SpendFromAccount#setAccountId(String)}
       * @return updated action object
       */
      public SpendFromAccount setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of asset to be spent
       * @param amount number of units of the asset to be spent
       * @return updated action object
       */
      public SpendFromAccount setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }

      /**
       * Specifies the time to live for this action.
       * @param ttlMS the ttl, in milliseconds
       * @return updated action object
       */
      public SpendFromAccount setTTL(long ttlMS) {
        this.put("ttl", ttlMS);
        return this;
      }
    }

    /**
     * Represents a control action taken on a particular account.
     */
    public static class ControlWithAccount extends Action {
      /**
       * Default constructor defines the action type as "control_account"
       */
      public ControlWithAccount() {
        this.put("type", "control_account");
      }

      /**
       * Specifies the controlling account using its alias.<br>
       * <strong>Must</strong> be used with {@link ControlWithAccount#setAssetAlias(String)}
       * @param alias alias of the controlling account
       * @return updated action object
       */
      public ControlWithAccount setAccountAlias(String alias) {
        this.put("account_alias", alias);
        return this;
      }

      /**
       * Specifies the controlling account using its id.<br>
       * <strong>Must</strong> be used with {@link ControlWithAccount#setAssetId(String)}
       * @param id id of the controlling account
       * @return updated action object
       */
      public ControlWithAccount setAccountId(String id) {
        this.put("account_id", id);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its alias.<br>
       * <strong>Must</strong> be used with {@link ControlWithAccount#setAccountAlias(String)}
       * @param alias alias of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its id.<br>
       * <strong>Must</strong> be used with {@link ControlWithAccount#setAccountId(String)}
       * @param id id of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.
       * @param amount number of units of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }
    }

    /**
     * Represents a control action taken on a control program.
     */
    public static class ControlWithProgram extends Action {
      /**
       * Default constructor defines the action type as "control_program"
       */
      public ControlWithProgram() {
        this.put("type", "control_program");
      }

      /**
       * Specifies the control program to be used.
       * @param controlProgram the control program to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(ControlProgram controlProgram) {
        this.put("control_program", controlProgram.program);
        return this;
      }

      /**
       * Specifies the control program to be used.
       * @param controlProgram the control program (as a string) to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(String controlProgram) {
        this.put("control_program", controlProgram);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its alias
       * @param alias alias of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its id
       * @param id id of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.
       * @param amount number of units of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }
    }

    /**
     * Represents a retire action.
     */
    public static class Retire extends Action {
      /**
       * Default constructor defines the action type as "control_program"
       */
      public Retire() {
        this.put("type", "control_program");
        this.put("control_program", ControlProgram.retireProgram());
      }

      /**
       * Specifies the amount of the asset to be retired.
       * @param amount number of units of the asset to be retired
       * @return updated action object
       */
      public Retire setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }

      /**
       * Specifies the asset to be retired using its alias
       * @param alias alias of the asset to be retired
       * @return updated action object
       */
      public Retire setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be retired using its id
       * @param id id of the asset to be retired
       * @return updated action object
       */
      public Retire setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }
    }

    /**
     * Sets the transaction-level reference data.
     * May only be used once per transaction.
     */
    public static class SetTransactionReferenceData extends Action {
      public SetTransactionReferenceData() {
        this.put("type", "set_transaction_reference_data");
      }

      public SetTransactionReferenceData(Map<String, Object> referenceData) {
        this();
        setReferenceData(referenceData);
      }

      /**
       * Adds a k,v pair to the action's reference data object.<br>
       * Since most/all current action types use the reference_data parameter, we provide this method in the base class to avoid repetition.
       * @param key key of the reference data field
       * @param value value of reference data field
       * @return updated SetTransactionReferenceData object
       */
      public Action addReferenceDataField(String key, Object value) {
        Map<String, Object> referenceData = (HashMap<String, Object>) this.get("reference_data");
        if (referenceData == null) {
          referenceData = new HashMap<>();
          this.put("reference_data", referenceData);
        }
        referenceData.put(key, value);
        return this;
      }

      /**
       * Specifies the reference data.<br>
       * @param referenceData reference data to embed into the action
       * @return updated SetTransactionReferenceData object
       */
      public SetTransactionReferenceData setReferenceData(Map<String, Object> referenceData) {
        this.put("reference_data", referenceData);
        return this;
      }
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
     * Hex-encoded serialization of a transaction to add to the current template.
     */
    @SerializedName("base_transaction")
    private String baseTransaction;

    /**
     * List of actions in a transaction.
     */
    private List<Action> actions;

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
      return ctx.singletonBatchRequest("build-transaction", Arrays.asList(this), Template.class);
    }

    /**
     * Default constructor initializes actions list.
     */
    public Builder() {
      this.actions = new ArrayList<>();
    }

    /**
     * Sets the baseTransaction field and initializes the actions lists.<br>
     * This constructor can be used when executing an atomic swap and the counter party has sent an initialized tx template.
     */
    public Builder(String rawTransaction) {
      this.baseTransaction = rawTransaction;
      this.actions = new ArrayList<>();
    }

    /**
     * Sets the rawTransaction that will be added to the current template.
     */
    public Builder setBaseTransaction(String rawTransaction) {
      this.baseTransaction = rawTransaction;
      return this;
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
  }

  /**
   * When used in conjunction with /list-transactions, Feeds can be used to
   * receive notifications about transactions.
   */
  public static class Feed {
    /**
     * Feed ID, automatically generated when a feed is created.
     */
    public String id;

    /**
     * An optional, user-supplied alias that can be used to uniquely identify
     * this feed.
     */
    public String alias;

    /**
     * The query filter used in /list-transactions.
     */
    public String filter;

    /**
     * Indicates the last transaction consumed by this feed.
     */
    public String after;

    private ListIterator<Transaction> txIter;
    private Transaction lastTx;

    /**
     * Creates a feed.
     *
     * @param ctx context object that makes requests to core
     * @param alias an alias which uniquely identifies this feed
     * @param filter a query filter which identifies which transactions this feed consumes
     * @return a feed object
     * @throws ChainException
     */
    public static Feed create(Context ctx, String alias, String filter) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      req.put("filter", filter);
      req.put("client_token", UUID.randomUUID().toString());
      return ctx.request("create-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves a feed by ID.
     *
     * @param ctx context object that makes requests to core
     * @param id the feed id
     * @return a feed object
     * @throws ChainException
     */
    public static Feed getByID(Context ctx, String id) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("id", id);
      return ctx.request("get-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves a feed by alias.
     *
     * @param ctx context object that makes requests to core
     * @param alias the feed alias
     * @return a feed object
     * @throws ChainException
     */
    public static Feed getByAlias(Context ctx, String alias) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      return ctx.request("get-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves the next transaction matching the feed's filter criteria.
     * If no such transaction is available, this method will block until a
     * matching transaction arrives in the blockchain, or if the specified
     * timeout is reached. To avoid client-side timeouts, be sure to call
     * {@link Context#setReadTimeout(long, TimeUnit)} with appropriate
     * parameters.
     *
     * @param ctx context object that makes requests to core
     * @param timeout number of milliseconds before the server-side long-poll should time out
     * @return a transaction object
     * @throws ChainException
     */
    public Transaction next(Context ctx, long timeout) throws ChainException {
      if (txIter == null || !txIter.hasNext()) {
        txIter =
            new QueryBuilder()
                .setFilter(filter)
                .setAfter(after)
                .setTimeout(timeout)
                .setAscendingWithLongPoll()
                .execute(ctx)
                .list
                .listIterator();
      }

      lastTx = txIter.next();
      return lastTx;
    }

    /**
     * Retrieves the next transaction matching the feed's filter criteria.
     * If no such transaction is available, this method will block until a
     * matching transaction arrives in the blockchain. To avoid client-side
     * timeouts, be sure to call {@link Context#setReadTimeout(long, TimeUnit)}
     * with appropriate parameters.
     *
     * @param ctx context object that makes requests to core
     * @return a transaction object
     * @throws ChainException
     */
    public Transaction next(Context ctx) throws ChainException {
      return next(ctx, 0);
    }

    /**
     * Persists the state of the transaction feed. Be sure to call this
     * periodically when consuming transactions with
     * {@link #next(Context, long)}. The most conservative (albeit least
     * performant) strategy is to call ack() once for every result returned by
     * {@link #next(Context, long)}.
     *
     * @param ctx context object that makes requests to core
     * @throws ChainException
     */
    public void ack(Context ctx) throws ChainException {
      if (lastTx == null) {
        return;
      }

      // The format of the cursor value is specified in the core/query package.
      // It technically uses an unsigned 64-bit int for the end specifier, but
      // Long.MAX_VALUE should suffice.
      String newAfter = "" + lastTx.blockHeight + ":" + lastTx.position + "-" + Long.MAX_VALUE;
      Map<String, Object> req = new HashMap<>();
      req.put("id", this.id);
      req.put("previous_after", this.after);
      req.put("after", newAfter);
      ctx.request("update-transaction-feed", req, Feed.class);

      this.after = newAfter;
    }
  }
}
