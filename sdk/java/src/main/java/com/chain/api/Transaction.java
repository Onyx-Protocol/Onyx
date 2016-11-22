package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;

import java.util.*;
import java.util.concurrent.TimeUnit;

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
      Items items = this.client.request("list-transactions", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  /**
   * Transaction.QueryBuilder utilizes the builder pattern to create {@link Transaction} queries.<br>
   * The possible parameters for each query can be found on this class as well as the {@link BaseQueryBuilder} class.<br>
   * All parameters are optional, and should be set to filter the results accordingly.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes a transaction query based on provided parameters.
     * @param client client object which makes server requests
     * @return a page of transactions
     * @throws APIException This exception is raised if the api returns errors while processing the query.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items execute(Client client) throws ChainException {
      Items items = new Items();
      items.setClient(client);
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
     * The type of the input.<br>
     * Possible values are "issue" and "spend".
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
     * The type the output.<br>
     * Possible values are "control" and "retire".
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
   * A built transaction that has not been submitted for block inclusion (returned from {@link Transaction#buildBatch(Client, List)}).
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
    private boolean allowAdditionalActions;

    /**
     * allowAdditionalActions causes the transaction to be signed so
     * that it can be used as a base transaction in a multiparty trade
     * flow. To enable this setting, call this method after building the
     * transaction, but before sending it to the signer.
     *
     * All participants in a multiparty trade flow should call this
     * method except for the last signer. Do not call this option if
     * the transaction is complete, i.e. if it will not be used as a
     * base transaction.
     * @return updated transaction template
     */
    public Template allowAdditionalActions() {
      this.allowAdditionalActions = true;
      return this;
    }

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
   * A single response from a call to {@link Transaction#submitBatch(Client, List)}
   */
  public static class SubmitResponse {
    /**
     * The transaction id.
     */
    public String id;
  }

  /**
   * Builds a batch of transaction templates.
   * @param client client object which makes server requests
   * @param builders list of transaction builders
   * @return a list of transaction templates
   * @throws APIException This exception is raised if the api returns errors while building transaction templates.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<Template> buildBatch(
      Client client, List<Transaction.Builder> builders) throws ChainException {
    return client.batchRequest("build-transaction", builders, Template.class, BuildException.class);
  }

  /**
   * Submits a batch of signed transaction templates for inclusion into a block.
   * @param client client object which makes server requests
   * @param templates list of transaction templates
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<SubmitResponse> submitBatch(Client client, List<Template> templates)
      throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", templates);
    return client.batchRequest(
        "submit-transaction", body, SubmitResponse.class, APIException.class);
  }

  /**
   * Submits a batch of signed transaction templates for inclusion into a block.
   * @param client client object which makes server requests
   * @param templates list of transaction templates
   * @param waitUntil when the server should wait until responding - none, confirmed, processed
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<SubmitResponse> submitBatch(
      Client client, List<Template> templates, String waitUntil) throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", templates);
    body.put("wait_until", waitUntil);
    return client.batchRequest(
        "submit-transaction", body, SubmitResponse.class, APIException.class);
  }

  /**
   * Submits signed transaction template for inclusion into a block.
   * @param client client object which makes server requests
   * @param template transaction template
   * @return submit responses
   * @throws APIException This exception is raised if the api returns errors while submitting a transaction.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static SubmitResponse submit(Client client, Template template) throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", Arrays.asList(template));
    return client.singletonBatchRequest(
        "submit-transaction", body, SubmitResponse.class, APIException.class);
  }

  /**
   * Submits signed transaction template for inclusion into a block.
   * @param client client object which makes server requests
   * @param template transaction template
   * @param waitUntil when the server should wait until responding - none, confirmed, processed
   * @return submit responses
   * @throws APIException This exception is raised if the api returns errors while submitting a transaction.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static SubmitResponse submit(Client client, Template template, String waitUntil)
      throws ChainException {
    HashMap<String, Object> body = new HashMap<>();
    body.put("transactions", Arrays.asList(template));
    body.put("wait_until", waitUntil);
    return client.singletonBatchRequest(
        "submit-transaction", body, SubmitResponse.class, APIException.class);
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
       * Specifies the asset to be issued using its alias.<br>
       * <strong>Either this or {@link Issue#setAssetId(String)}  must be called.</strong>
       * @param alias alias of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be issued using its id.<br>
       * <strong>Either this or {@link Issue#setAssetAlias(String)} must be called.</strong>
       * @param id id of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be issued.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be issued
       * @return updated action object
       */
      public Issue setAmount(long amount) {
        this.put("amount", amount);
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
       * Specifies the unspent output to be spent.<br>
       * <strong>Either this or a combination of {@link SpendAccountUnspentOutput#setTransactionId(String)}
       * and {@link SpendAccountUnspentOutput#setPosition(int)} must be called.</strong>
       * @param unspentOutput unspent output to be spent
       * @return updated action object
       */
      public SpendAccountUnspentOutput setUnspentOutput(UnspentOutput unspentOutput) {
        setTransactionId(unspentOutput.transactionId);
        setPosition(unspentOutput.position);
        return this;
      }

      /**
       * Specifies the transaction id of the unspent output to be spent.<br>
       * <strong>Must be called with {@link SpendAccountUnspentOutput#setPosition(int)}.</strong>
       * @param id
       * @return
       */
      public SpendAccountUnspentOutput setTransactionId(String id) {
        this.put("transaction_id", id);
        return this;
      }

      /**
       * Specifies the position in the transaction of the unspent output to be spent.<br>
       * <strong>Must be called with {@link SpendAccountUnspentOutput#setTransactionId(String)}.</strong>
       * @param pos
       * @return
       */
      public SpendAccountUnspentOutput setPosition(int pos) {
        this.put("position", pos);
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
       * <strong>Either this or {@link SpendFromAccount#setAccountId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link SpendFromAccount#setAssetAlias(String)}.</strong>
       * @param alias alias of the spending account
       * @return updated action object
       */
      public SpendFromAccount setAccountAlias(String alias) {
        this.put("account_alias", alias);
        return this;
      }

      /**
       * Specifies the spending account using its id.<br>
       * <strong>Either this or {@link SpendFromAccount#setAccountAlias(String)} must be called.</strong><br>
       * <strong>Must be used with {@link SpendFromAccount#setAssetId(String)}.</strong>
       * @param id id of the spending account
       * @return updated action object
       */
      public SpendFromAccount setAccountId(String id) {
        this.put("account_id", id);
        return this;
      }

      /**
       * Specifies the asset to be spent using its alias.<br>
       * <strong>Either this or {@link SpendFromAccount#setAssetId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link SpendFromAccount#setAccountAlias(String)}.</strong>
       * @param alias alias of the asset to be spent
       * @return updated action object
       */
      public SpendFromAccount setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be spent using its id.<br>
       * <strong>Either this or {@link SpendFromAccount#setAssetAlias(String)} must be called.</strong><br>
       * <strong>Must be used with {@link SpendFromAccount#setAccountId(String)}.</strong><br>
       * @param id id of the asset to be spent
       * @return updated action object
       */
      public SpendFromAccount setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of asset to be spent.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be spent
       * @return updated action object
       */
      public SpendFromAccount setAmount(long amount) {
        this.put("amount", amount);
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
       * <strong>Either this or {@link ControlWithAccount#setAccountId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link ControlWithAccount#setAssetAlias(String)}.</strong>
       * @param alias alias of the controlling account
       * @return updated action object
       */
      public ControlWithAccount setAccountAlias(String alias) {
        this.put("account_alias", alias);
        return this;
      }

      /**
       * Specifies the controlling account using its id.<br>
       * <strong>Either this or {@link ControlWithAccount#setAccountAlias(String)} must be called.</strong><br>
       * <strong>Must be used with {@link ControlWithAccount#setAssetId(String)}.</strong>
       * @param id id of the controlling account
       * @return updated action object
       */
      public ControlWithAccount setAccountId(String id) {
        this.put("account_id", id);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its alias.<br>
       * <strong>Either this or {@link ControlWithAccount#setAssetId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link ControlWithAccount#setAccountAlias(String)}.</strong>
       * @param alias alias of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its id.<br>
       * <strong>Either this or {@link ControlWithAccount#setAssetAlias(String)} must be called.</strong><br>
       * <strong>Must be used with {@link ControlWithAccount#setAccountId(String)}.</strong>
       * @param id id of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.<br>
       * <strong>Must be called.</strong>
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
       * Specifies the control program to be used.<br>
       * <strong>Either this or {@link ControlWithProgram#setControlProgram(String)} must be called.</strong>
       * @param controlProgram the control program to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(ControlProgram controlProgram) {
        this.put("control_program", controlProgram.controlProgram);
        return this;
      }

      /**
       * Specifies the control program to be used.<br>
       * <strong>Either this or {@link ControlWithProgram#setControlProgram(ControlProgram)} must be called.</strong>
       * @param controlProgram the control program (as a string) to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(String controlProgram) {
        this.put("control_program", controlProgram);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its alias.<br>
       * <strong>Either this or {@link ControlWithProgram#setAssetId(String)} must be called.</strong>
       * @param alias alias of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be controlled using its id.<br>
       * <strong>Either this or {@link ControlWithProgram#setAssetAlias(String)} must be called.</strong>
       * @param id id of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetId(String id) {
        this.put("asset_id", id);
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.<br>
       * <strong>Must be called.</strong>
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
       * Specifies the amount of the asset to be retired.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be retired
       * @return updated action object
       */
      public Retire setAmount(long amount) {
        this.put("amount", amount);
        return this;
      }

      /**
       * Specifies the asset to be retired using its alias.<br>
       * <strong>Either this or {@link Retire#setAssetId(String)}  must be called.</strong>
       * @param alias alias of the asset to be retired
       * @return updated action object
       */
      public Retire setAssetAlias(String alias) {
        this.put("asset_alias", alias);
        return this;
      }

      /**
       * Specifies the asset to be retired using its id.<br>
       * <strong>Either this or {@link Retire#setAssetAlias(String)} must be called.</strong>
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
   * Transaction.Builder utilizes the builder pattern to create {@link Transaction.Template} objects.
   * At minimum, a {@link Action.Issue} or {@link Action.SpendFromAccount}/{@link Action.SpendAccountUnspentOutput}
   * must be coupled with a {@link Action.ControlWithAccount}/{@link Action.ControlWithProgram} before calling {@link #build(Client)}.
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
     * A time duration in milliseconds. If the transaction is not fully
     * signed and submitted within this time, it will be rejected by the
     * blockchain. Additionally, any outputs reserved when building this
     * transaction will remain reserved for this duration.
     */
    private long ttl;

    /**
     * Builds a single transaction template.
     * @param client client object which makes requests to the server
     * @return a transaction template
     * @throws APIException This exception is raised if the api returns errors while building the transaction.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Template build(Client client) throws ChainException {
      return client.singletonBatchRequest(
          "build-transaction", Arrays.asList(this), Template.class, BuildException.class);
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
    public Builder(String baseTransaction) {
      this.setBaseTransaction(baseTransaction);
      this.actions = new ArrayList<>();
    }

    /**
     * Sets the base transaction that will be added to the current template.
     */
    public Builder setBaseTransaction(String baseTransaction) {
      this.baseTransaction = baseTransaction;
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
     * @param client client object that makes requests to core
     * @param alias an alias which uniquely identifies this feed
     * @param filter a query filter which identifies which transactions this feed consumes
     * @return a feed object
     * @throws ChainException
     */
    public static Feed create(Client client, String alias, String filter) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      req.put("filter", filter);
      req.put("client_token", UUID.randomUUID().toString());
      return client.request("create-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves a feed by ID.
     *
     * @param client client object that makes requests to core
     * @param id the feed id
     * @return a feed object
     * @throws ChainException
     */
    public static Feed getByID(Client client, String id) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("id", id);
      return client.request("get-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves a feed by alias.
     *
     * @param client client object that makes requests to core
     * @param alias the feed alias
     * @return a feed object
     * @throws ChainException
     */
    public static Feed getByAlias(Client client, String alias) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      return client.request("get-transaction-feed", req, Feed.class);
    }

    /**
     * Retrieves the next transaction matching the feed's filter criteria.
     * If no such transaction is available, this method will block until a
     * matching transaction arrives in the blockchain, or if the specified
     * timeout is reached. To avoid client-side timeouts, be sure to call
     * {@link Client#setReadTimeout(long, TimeUnit)} (long, TimeUnit)} with appropriate
     * parameters.
     *
     * @param client client object that makes requests to core
     * @param timeout number of milliseconds before the server-side long-poll should time out
     * @return a transaction object
     * @throws ChainException
     */
    public Transaction next(Client client, long timeout) throws ChainException {
      if (txIter == null || !txIter.hasNext()) {
        txIter =
            new QueryBuilder()
                .setFilter(filter)
                .setAfter(after)
                .setTimeout(timeout)
                .setAscendingWithLongPoll()
                .execute(client)
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
     * timeouts, be sure to call {@link Client#setReadTimeout(long, TimeUnit)}
     * with appropriate parameters.
     *
     * @param client client object that makes requests to core
     * @return a transaction object
     * @throws ChainException
     */
    public Transaction next(Client client) throws ChainException {
      return next(client, 0);
    }

    /**
     * Persists the state of the transaction feed. Be sure to call this
     * periodically when consuming transactions with
     * {@link #next(Client, long)}. The most conservative (albeit least
     * performant) strategy is to call ack() once for every result returned by
     * {@link #next(Client, long)}.
     *
     * @param client context object that makes requests to core
     * @throws ChainException
     */
    public void ack(Client client) throws ChainException {
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
      client.request("update-transaction-feed", req, Feed.class);

      this.after = newAfter;
    }
  }
}
