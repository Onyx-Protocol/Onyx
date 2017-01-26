package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;
import com.chain.proto.*;
import com.google.common.reflect.TypeToken;
import com.google.gson.annotations.SerializedName;
import com.google.protobuf.ByteString;

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
  public static class Items extends PagedItems<Transaction, ListTxsQuery> {
    /**
     * Returns a new page of transactions based on the underlying query.
     * @return a page of transactions
     * @throws APIException This exception is raised if the api returns errors while processing the query.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    @Override
    public Items getPage() throws ChainException {
      ListTxsResponse resp = this.client.app().listTxs(this.next);
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }
      Items items = new Items();
      items.list =
          this.client.deserialize(
              new String(resp.getItems().toByteArray()),
              new TypeToken<List<Transaction>>() {}.getType());
      items.lastPage = resp.getLastPage();
      items.next = resp.getNext();
      items.setClient(this.client);
      return items;
    }

    public void setNext(Query query) {
      ListTxsQuery.Builder builder =
          ListTxsQuery.newBuilder()
              .setAscendingWithLongPoll(query.ascendingWithLongPoll)
              .setStartTime(query.startTime)
              .setEndTime(query.endTime);
      if (query.timeout > 0) {
        builder.setTimeout(Long.valueOf(query.timeout).toString() + "ms");
      }
      if (query.filter != null && !query.filter.isEmpty()) {
        builder.setFilter(query.filter);
      }
      if (query.after != null && !query.after.isEmpty()) {
        builder.setAfter(query.after);
      }

      if (query.filterParams != null) {
        for (Query.FilterParam param : query.filterParams) {
          builder.addFilterParams(param.toProtobuf());
        }
      }

      this.next = builder.build();
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
    public byte[] rawTransaction;

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

    public Template() {}

    public Template(TxTemplate proto) {
      this.rawTransaction = proto.getRawTransaction().toByteArray();
      this.signingInstructions =
          SigningInstruction.fromProtobuf(proto.getSigningInstructionsList());
      this.local = proto.getLocal();
      this.allowAdditionalActions = proto.getAllowAdditionalActions();
    }

    public TxTemplate toProtobuf() {
      TxTemplate.Builder tpl = TxTemplate.newBuilder();
      if (this.rawTransaction != null) {
        tpl.setRawTransaction(ByteString.copyFrom(this.rawTransaction));
      }
      if (this.signingInstructions != null) {
        for (SigningInstruction sig : this.signingInstructions) {
          tpl.addSigningInstructions(sig.toProtobuf());
        }
      }
      tpl.setLocal(this.local);
      tpl.setAllowAdditionalActions(this.allowAdditionalActions);
      return tpl.build();
    }

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
      public byte[] assetID;

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

      private SigningInstruction(TxTemplate.SigningInstruction proto) {
        this.position = proto.getPosition();
        this.assetID = proto.getAssetId().toByteArray();
        this.amount = proto.getAmount();
        this.witnessComponents = WitnessComponent.fromProtobuf(proto.getWitnessComponentsList());
      }

      private static List<SigningInstruction> fromProtobuf(
          List<TxTemplate.SigningInstruction> protos) {
        ArrayList<SigningInstruction> sigs = new ArrayList();
        for (TxTemplate.SigningInstruction proto : protos) {
          sigs.add(new SigningInstruction(proto));
        }
        return sigs;
      }

      private TxTemplate.SigningInstruction toProtobuf() {
        TxTemplate.SigningInstruction.Builder proto = TxTemplate.SigningInstruction.newBuilder();
        if (this.assetID != null) {
          proto.setAssetId(ByteString.copyFrom(this.assetID));
        }
        proto.setAmount(this.amount);
        proto.setPosition(this.position);
        if (this.witnessComponents != null) {
          for (WitnessComponent comp : this.witnessComponents) {
            proto.addWitnessComponents(comp.toProtobuf());
          }
        }
        return proto.build();
      }
    }

    /**
     * A single witness component, holding information that will become the input witness.
     */
    public static class WitnessComponent {
      /**
       * The type of witness component.<br>
       * Possible types are "signature".
       */
      public String type;

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
      public byte[] program;

      /**
       * The list of signatures made with the specified keys (null unless type is "signature").
       */
      public byte[][] signatures;

      private WitnessComponent(TxTemplate.WitnessComponent proto) {
        switch (proto.getComponentCase()) {
          case SIGNATURE:
            this.type = "signature";
            TxTemplate.SignatureComponent sigComp = proto.getSignature();
            this.quorum = sigComp.getQuorum();
            this.keys = KeyID.fromProtobuf(sigComp.getKeyIdsList());
            this.program = sigComp.getProgram().toByteArray();
            this.signatures = new byte[sigComp.getSignaturesCount()][];
            for (int i = 0; i < sigComp.getSignaturesCount(); i++) {
              this.signatures[i] = sigComp.getSignatures(i).toByteArray();
            }
        }
      }

      private static WitnessComponent[] fromProtobuf(List<TxTemplate.WitnessComponent> protos) {
        WitnessComponent[] comps = new WitnessComponent[protos.size()];
        for (int i = 0; i < protos.size(); i++) {
          comps[i] = new WitnessComponent(protos.get(i));
        }
        return comps;
      }

      private TxTemplate.WitnessComponent toProtobuf() {
        TxTemplate.WitnessComponent.Builder proto = TxTemplate.WitnessComponent.newBuilder();

        switch (this.type) {
          case "signature":
            TxTemplate.SignatureComponent.Builder sigComp =
                TxTemplate.SignatureComponent.newBuilder();
            sigComp.setQuorum(this.quorum);
            if (this.program != null) {
              sigComp.setProgram(ByteString.copyFrom(this.program));
            }
            if (this.signatures != null) {
              for (byte[] sig : this.signatures) {
                sigComp.addSignatures(ByteString.copyFrom(sig));
              }
            }
            if (this.keys != null) {
              for (KeyID key : this.keys) {
                sigComp.addKeyIds(key.toProtobuf());
              }
            }
            proto.setSignature(sigComp);
        }

        return proto.build();
      }
    }

    /**
     * A class representing a derived signing key.
     */
    public static class KeyID {
      /**
       * The extended public key associated with the private key used to sign.
       */
      public byte[] xpub;

      /**
       * The derivation path of the extended public key.
       */
      @SerializedName("derivation_path")
      public byte[][] derivationPath;

      private KeyID(TxTemplate.KeyID proto) {
        this.xpub = proto.getXpub().toByteArray();
        this.derivationPath = new byte[proto.getDerivationPathCount()][];
        for (int i = 0; i < proto.getDerivationPathCount(); i++) {
          this.derivationPath[i] = proto.getDerivationPath(i).toByteArray();
        }
      }

      private static KeyID[] fromProtobuf(List<TxTemplate.KeyID> protos) {
        KeyID[] keys = new KeyID[protos.size()];
        for (int i = 0; i < protos.size(); i++) {
          keys[i] = new KeyID(protos.get(i));
        }
        return keys;
      }

      private TxTemplate.KeyID toProtobuf() {
        TxTemplate.KeyID.Builder proto = TxTemplate.KeyID.newBuilder();
        if (this.xpub != null) {
          proto.setXpub(ByteString.copyFrom(this.xpub));
        }

        if (this.derivationPath != null) {
          for (byte[] path : this.derivationPath) {
            proto.addDerivationPath(ByteString.copyFrom(path));
          }
        }

        return proto.build();
      }
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
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<Template> buildBatch(
      Client client, List<Transaction.Builder> builders) throws ChainException {
    ArrayList<BuildTxsRequest.Request> reqs = new ArrayList();
    for (Transaction.Builder builder : builders) {
      BuildTxsRequest.Request.Builder req = BuildTxsRequest.Request.newBuilder();
      if (builder.ttl > 0) {
        req.setTtl(Long.valueOf(builder.ttl).toString() + "ms");
      }
      if (builder.baseTransaction != null) {
        req.setTransaction(ByteString.copyFrom(builder.baseTransaction));
      }
      if (builder.actions != null) {
        for (Action action : builder.actions) {
          req.addActions(action.toProtobuf(client));
        }
      }
      reqs.add(req.build());
    }

    BuildTxsRequest req = BuildTxsRequest.newBuilder().addAllRequests(reqs).build();
    TxsResponse resp = client.app().buildTxs(req);
    if (resp.hasError()) {
      throw new APIException(resp.getError());
    }

    Map<Integer, Template> successes = new LinkedHashMap();
    Map<Integer, APIException> errors = new LinkedHashMap();

    for (int i = 0; i < resp.getResponsesCount(); i++) {
      TxsResponse.Response r = resp.getResponses(i);
      if (r.hasError()) {
        errors.put(i, new APIException(r.getError()));
      } else {
        successes.put(i, new Template(r.getTemplate()));
      }
    }

    return new BatchResponse<Template>(successes, errors);
  }

  /**
   * Submits a batch of signed transaction templates for inclusion into a block.
   * @param client client object which makes server requests
   * @param templates list of transaction templates
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<SubmitResponse> submitBatch(Client client, List<Template> templates)
      throws ChainException {
    return submitBatch(client, templates, null);
  }

  /**
   * Submits a batch of signed transaction templates for inclusion into a block.
   * @param client client object which makes server requests
   * @param templates list of transaction templates
   * @param waitUntil when the server should wait until responding - none, confirmed, processed
   * @return a list of submit responses (individual objects can hold transaction ids or error info)
   * @throws APIException This exception is raised if the api returns errors while submitting transactions.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<SubmitResponse> submitBatch(
      Client client, List<Template> templates, String waitUntil) throws ChainException {
    SubmitTxsRequest.Builder req = SubmitTxsRequest.newBuilder();
    if (waitUntil != null) {
      req.setWaitUntil(waitUntil);
    }
    for (Template template : templates) {
      req.addTransactions(template.toProtobuf());
    }

    SubmitTxsResponse resp = client.app().submitTxs(req.build());
    if (resp.hasError()) {
      throw new APIException(resp.getError());
    }

    Map<Integer, SubmitResponse> successes = new LinkedHashMap();
    Map<Integer, APIException> errors = new LinkedHashMap();

    for (int i = 0; i < resp.getResponsesCount(); i++) {
      SubmitTxsResponse.Response r = resp.getResponses(i);
      if (r.hasError()) {
        errors.put(i, new APIException(r.getError()));
      } else {
        SubmitResponse sr = new SubmitResponse();
        sr.id = r.getId();
        successes.put(i, sr);
      }
    }

    return new BatchResponse<SubmitResponse>(successes, errors);
  }

  /**
   * Submits signed transaction template for inclusion into a block.
   * @param client client object which makes server requests
   * @param template transaction template
   * @return submit responses
   * @throws APIException This exception is raised if the api returns errors while submitting a transaction.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static SubmitResponse submit(Client client, Template template) throws ChainException {
    return submit(client, template, "");
  }

  /**
   * Submits signed transaction template for inclusion into a block.
   * @param client client object which makes server requests
   * @param template transaction template
   * @param waitUntil when the server should wait until responding - none, confirmed, processed
   * @return submit responses
   * @throws APIException This exception is raised if the api returns errors while submitting a transaction.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static SubmitResponse submit(Client client, Template template, String waitUntil)
      throws ChainException {
    BatchResponse<SubmitResponse> resp = Transaction.submitBatch(client, Arrays.asList(template));
    if (resp.isError(0)) {
      throw resp.errorsByIndex().get(0);
    }
    return resp.successesByIndex().get(0);
  }

  /**
   * Base class representing actions that can be taken within a transaction.
   */
  abstract public static class Action {

    protected String clientToken;
    protected Map<String, Object> referenceData;
    /**
     * Default constructor initializes list and sets the client token.
     */
    public Action() {
      // Several action types require client_token as an idempotency key.
      // It's safest to include a default value for this param.
      clientToken = UUID.randomUUID().toString();
    }

    /**
     * Adds a k,v pair to the action's reference data object.<br>
     * Since most/all current action types use the reference_data parameter, we provide this method in the base class to avoid repetition.
     * @param key key of the reference data field
     * @param value value of reference data field
     * @return updated action object
     */
    public Action addReferenceDataField(String key, Object value) {
      if (referenceData == null) {
        referenceData = new HashMap<>();
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
      this.referenceData = referenceData;
      return this;
    }

    abstract com.chain.proto.Action toProtobuf(Client client);

    /**
     * Represents an issuance action.
     */
    public static class Issue extends Action {

      private String assetAlias;
      private byte[] assetID;
      private long amount;

      /**
       * Default constructor defines the action type as "issue"
       */
      public Issue() {
        super();
      }

      /**
       * Specifies the asset to be issued using its alias.<br>
       * <strong>Either this or {@link Issue#setAssetId(byte[])}  must be called.</strong>
       * @param alias alias of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetAlias(String alias) {
        assetAlias = alias;
        return this;
      }

      /**
       * Specifies the asset to be issued using its id.<br>
       * <strong>Either this or {@link Issue#setAssetAlias(String)} must be called.</strong>
       * @param id id of the asset to be issued
       * @return updated action object
       */
      public Issue setAssetId(String id) {
        assetID = Util.hexStringToByteArray(id);
        return this;
      }

      public Issue setAssetId(byte[] id) {
        assetID = id;
        return this;
      }

      /**
       * Specifies the amount of the asset to be issued.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be issued
       * @return updated action object
       */
      public Issue setAmount(long amount) {
        this.amount = amount;
        return this;
      }

      @Override
      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.Issue.Builder builder = com.chain.proto.Action.Issue.newBuilder();
        builder.setAmount(amount);
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }
        AssetIdentifier.Builder id = AssetIdentifier.newBuilder();
        if (assetID != null) {
          id.setAssetId(ByteString.copyFrom(assetID));
        } else if (assetAlias != null && !assetAlias.isEmpty()) {
          id.setAssetAlias(assetAlias);
        }
        builder.setAsset(id);

        return com.chain.proto.Action.newBuilder().setIssue(builder).build();
      }
    }

    /**
     * Represents a spend action taken on a particular unspent output.
     */
    public static class SpendAccountUnspentOutput extends Action {
      private byte[] transactionID;
      private int position;

      /**
       * Default constructor defines the action type as "spend_account_unspent_output"
       */
      public SpendAccountUnspentOutput() {
        super();
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
        transactionID = Util.hexStringToByteArray(id);
        return this;
      }

      public SpendAccountUnspentOutput setTransactionId(byte[] id) {
        transactionID = id;
        return this;
      }

      /**
       * Specifies the position in the transaction of the unspent output to be spent.<br>
       * <strong>Must be called with {@link SpendAccountUnspentOutput#setTransactionId(String)}.</strong>
       * @param pos
       * @return
       */
      public SpendAccountUnspentOutput setPosition(int pos) {
        position = pos;
        return this;
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.SpendAccountUnspentOutput.Builder builder =
            com.chain.proto.Action.SpendAccountUnspentOutput.newBuilder();
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }
        builder.setClientToken(clientToken);
        if (transactionID != null) {
          builder.setTxId(ByteString.copyFrom(transactionID));
        }
        builder.setPosition(position);

        return com.chain.proto.Action.newBuilder().setSpendAccountUnspentOutput(builder).build();
      }
    }

    /**
     * Represents a spend action taken on a particular account.
     */
    public static class SpendFromAccount extends Action {
      private String accountID;
      private String accountAlias;
      private byte[] assetID;
      private String assetAlias;
      private long amount;

      /**
       * Default constructor defines the action type as "spend_account"
       */
      public SpendFromAccount() {
        super();
      }

      /**
       * Specifies the spending account using its alias.<br>
       * <strong>Either this or {@link SpendFromAccount#setAccountId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link SpendFromAccount#setAssetAlias(String)}.</strong>
       * @param alias alias of the spending account
       * @return updated action object
       */
      public SpendFromAccount setAccountAlias(String alias) {
        accountAlias = alias;
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
        accountID = id;
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
        assetAlias = alias;
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
        assetID = Util.hexStringToByteArray(id);
        return this;
      }

      public SpendFromAccount setAssetId(byte[] id) {
        assetID = id;
        return this;
      }

      /**
       * Specifies the amount of asset to be spent.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be spent
       * @return updated action object
       */
      public SpendFromAccount setAmount(long amount) {
        this.amount = amount;
        return this;
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.SpendAccount.Builder builder =
            com.chain.proto.Action.SpendAccount.newBuilder();
        builder.setAmount(amount);
        builder.setClientToken(clientToken);
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }

        AssetIdentifier.Builder assetID = AssetIdentifier.newBuilder();
        if (this.assetID != null) {
          assetID.setAssetId(ByteString.copyFrom(this.assetID));
        } else if (assetAlias != null && !assetAlias.isEmpty()) {
          assetID.setAssetAlias(assetAlias);
        }
        builder.setAsset(assetID);

        AccountIdentifier.Builder accountID = AccountIdentifier.newBuilder();
        if (this.accountID != null && !this.accountID.isEmpty()) {
          accountID.setAccountId(this.accountID);
        } else if (accountAlias != null && !accountAlias.isEmpty()) {
          accountID.setAccountAlias(accountAlias);
        }
        builder.setAccount(accountID);

        return com.chain.proto.Action.newBuilder().setSpendAccount(builder).build();
      }
    }

    /**
     * Represents a control action taken on a particular account.
     */
    public static class ControlWithAccount extends Action {

      private String accountID;
      private String accountAlias;
      private byte[] assetID;
      private String assetAlias;
      private long amount;

      /**
       * Default constructor defines the action type as "control_account"
       */
      public ControlWithAccount() {
        super();
      }

      /**
       * Specifies the controlling account using its alias.<br>
       * <strong>Either this or {@link ControlWithAccount#setAccountId(String)} must be called.</strong><br>
       * <strong>Must be used with {@link ControlWithAccount#setAssetAlias(String)}.</strong>
       * @param alias alias of the controlling account
       * @return updated action object
       */
      public ControlWithAccount setAccountAlias(String alias) {
        accountAlias = alias;
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
        accountID = id;
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
        assetAlias = alias;
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
        assetID = Util.hexStringToByteArray(id);
        return this;
      }

      public ControlWithAccount setAssetId(byte[] id) {
        assetID = id;
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be controlled
       * @return updated action object
       */
      public ControlWithAccount setAmount(long amount) {
        this.amount = amount;
        return this;
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.ControlAccount.Builder builder =
            com.chain.proto.Action.ControlAccount.newBuilder();
        builder.setAmount(amount);
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }

        AssetIdentifier.Builder assetID = AssetIdentifier.newBuilder();
        if (this.assetID != null) {
          assetID.setAssetId(ByteString.copyFrom(this.assetID));
        } else if (assetAlias != null && !assetAlias.isEmpty()) {
          assetID.setAssetAlias(assetAlias);
        }
        builder.setAsset(assetID);

        AccountIdentifier.Builder accountID = AccountIdentifier.newBuilder();
        if (this.accountID != null && !this.accountID.isEmpty()) {
          accountID.setAccountId(this.accountID);
        } else if (accountAlias != null && !accountAlias.isEmpty()) {
          accountID.setAccountAlias(accountAlias);
        }
        builder.setAccount(accountID);

        return com.chain.proto.Action.newBuilder().setControlAccount(builder).build();
      }
    }

    /**
     * Represents a control action taken on a control program.
     */
    public static class ControlWithProgram extends Action {
      private byte[] controlProgram;
      private byte[] assetID;
      private String assetAlias;
      private long amount;

      /**
       * Default constructor defines the action type as "control_program"
       */
      public ControlWithProgram() {
        super();
      }

      /**
       * Specifies the control program to be used.<br>
       * <strong>Either this or {@link ControlWithProgram#setControlProgram(String)} must be called.</strong>
       * @param controlProgram the control program to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(ControlProgram controlProgram) {
        this.controlProgram = controlProgram.controlProgram;
        return this;
      }

      /**
       * Specifies the control program to be used.<br>
       * <strong>Either this or {@link ControlWithProgram#setControlProgram(ControlProgram)} must be called.</strong>
       * @param controlProgram the control program (as a string) to be used
       * @return updated action object
       */
      public ControlWithProgram setControlProgram(String controlProgram) {
        this.controlProgram = Util.hexStringToByteArray(controlProgram);
        return this;
      }

      public ControlWithProgram setControlProgram(byte[] controlProgram) {
        this.controlProgram = controlProgram;
        return this;
      }

      /**
       * Specifies the asset to be controlled using its alias.<br>
       * <strong>Either this or {@link ControlWithProgram#setAssetId(String)} must be called.</strong>
       * @param alias alias of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetAlias(String alias) {
        assetAlias = alias;
        return this;
      }

      /**
       * Specifies the asset to be controlled using its id.<br>
       * <strong>Either this or {@link ControlWithProgram#setAssetAlias(String)} must be called.</strong>
       * @param id id of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAssetId(String id) {
        assetID = Util.hexStringToByteArray(id);
        return this;
      }

      public ControlWithProgram setAssetId(byte[] id) {
        assetID = id;
        return this;
      }

      /**
       * Specifies the amount of the asset to be controlled.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be controlled
       * @return updated action object
       */
      public ControlWithProgram setAmount(long amount) {
        this.amount = amount;
        return this;
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.ControlProgram.Builder builder =
            com.chain.proto.Action.ControlProgram.newBuilder();
        builder.setAmount(amount);
        if (controlProgram != null) {
          builder.setControlProgram(ByteString.copyFrom(controlProgram));
        }
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }

        AssetIdentifier.Builder assetID = AssetIdentifier.newBuilder();
        if (this.assetID != null) {
          assetID.setAssetId(ByteString.copyFrom(this.assetID));
        } else if (assetAlias != null && !assetAlias.isEmpty()) {
          assetID.setAssetAlias(assetAlias);
        }
        builder.setAsset(assetID);

        return com.chain.proto.Action.newBuilder().setControlProgram(builder).build();
      }
    }

    /**
     * Represents a retire action.
     */
    public static class Retire extends Action {
      private long amount;
      private byte[] assetID;
      private String assetAlias;

      /**
       * Default constructor defines the action type as "control_program"
       */
      public Retire() {
        super();
      }

      /**
       * Specifies the amount of the asset to be retired.<br>
       * <strong>Must be called.</strong>
       * @param amount number of units of the asset to be retired
       * @return updated action object
       */
      public Retire setAmount(long amount) {
        this.amount = amount;
        return this;
      }

      /**
       * Specifies the asset to be retired using its alias.<br>
       * <strong>Either this or {@link Retire#setAssetId(String)}  must be called.</strong>
       * @param alias alias of the asset to be retired
       * @return updated action object
       */
      public Retire setAssetAlias(String alias) {
        assetAlias = alias;
        return this;
      }

      /**
       * Specifies the asset to be retired using its id.<br>
       * <strong>Either this or {@link Retire#setAssetAlias(String)} must be called.</strong>
       * @param id id of the asset to be retired
       * @return updated action object
       */
      public Retire setAssetId(String id) {
        assetID = Util.hexStringToByteArray(id);
        return this;
      }

      public Retire setAssetId(byte[] id) {
        assetID = id;
        return this;
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.ControlProgram.Builder builder =
            com.chain.proto.Action.ControlProgram.newBuilder();
        builder.setAmount(amount);
        builder.setControlProgram(ByteString.copyFrom(ControlProgram.retireProgram()));
        if (referenceData != null) {
          builder.setReferenceData(ByteString.copyFrom(client.serialize(referenceData)));
        }

        AssetIdentifier.Builder assetID = AssetIdentifier.newBuilder();
        if (this.assetID != null) {
          assetID.setAssetId(ByteString.copyFrom(this.assetID));
        } else if (assetAlias != null && !assetAlias.isEmpty()) {
          assetID.setAssetAlias(assetAlias);
        }
        builder.setAsset(assetID);

        return com.chain.proto.Action.newBuilder().setControlProgram(builder).build();
      }
    }

    /**
     * Sets the transaction-level reference data.
     * May only be used once per transaction.
     */
    public static class SetTransactionReferenceData extends Action {
      public SetTransactionReferenceData() {
        super();
      }

      public SetTransactionReferenceData(Map<String, Object> referenceData) {
        this();
        setReferenceData(referenceData);
      }

      protected com.chain.proto.Action toProtobuf(Client client) {
        com.chain.proto.Action.SetTxReferenceData.Builder builder =
            com.chain.proto.Action.SetTxReferenceData.newBuilder();
        if (referenceData != null) {
          builder.setData(ByteString.copyFrom(client.serialize(referenceData)));
        }

        return com.chain.proto.Action.newBuilder().setSetTxReferenceData(builder).build();
      }
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
    private byte[] baseTransaction;

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
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Template build(Client client) throws ChainException {
      BatchResponse<Template> resp = Transaction.buildBatch(client, Arrays.asList(this));
      if (resp.isError(0)) {
        throw resp.errorsByIndex().get(0);
      }
      return resp.successesByIndex().get(0);
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
    public Builder(byte[] baseTransaction) {
      this.setBaseTransaction(baseTransaction);
      this.actions = new ArrayList<>();
    }

    /**
     * Sets the base transaction that will be added to the current template.
     */
    public Builder setBaseTransaction(byte[] baseTransaction) {
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

    private Feed(TxFeed proto) {
      this.id = proto.getId();
      this.alias = proto.getAlias();
      this.filter = proto.getFilter();
      this.after = proto.getAfter();
    }

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
      CreateTxFeedRequest.Builder req = CreateTxFeedRequest.newBuilder();
      req.setClientToken(UUID.randomUUID().toString());
      if (alias != null && !alias.isEmpty()) {
        req.setAlias(alias);
      }
      if (filter != null && !filter.isEmpty()) {
        req.setFilter(filter);
      }

      TxFeedResponse resp = client.app().createTxFeed(req.build());
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      return new Feed(resp.getResponse());
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
      TxFeedResponse resp = client.app().getTxFeed(GetTxFeedRequest.newBuilder().setId(id).build());
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      return new Feed(resp.getResponse());
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
      TxFeedResponse resp =
          client.app().getTxFeed(GetTxFeedRequest.newBuilder().setAlias(alias).build());
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      return new Feed(resp.getResponse());
    }

    /**
     * Retrieves the next transaction matching the feed's filter criteria.
     * If no such transaction is available, this method will block until a
     * matching transaction arrives in the blockchain, or if the specified
     * timeout is reached.
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
     * matching transaction arrives in the blockchain.
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

      UpdateTxFeedRequest req =
          UpdateTxFeedRequest.newBuilder()
              .setId(this.id)
              .setPreviousAfter(this.after)
              .setAfter(newAfter)
              .build();

      TxFeedResponse resp = client.app().updateTxFeed(req);
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      this.after = newAfter;
    }
  }
}
