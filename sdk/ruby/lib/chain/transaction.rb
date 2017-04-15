require 'securerandom'

require_relative './client_module'
require_relative './query'
require_relative './response_object'

module Chain
  class Transaction < ResponseObject

    # @!attribute [r] id
    # Unique transaction identifier.
    # @return [String]
    attrib :id

    # @!attribute [r] timestamp
    # Time of transaction.
    # @return [Time]
    attrib :timestamp, rfc3339_time: true

    # @!attribute [r] block_id
    # Unique identifier, or block hash, of the block containing a transaction.
    # @return [String]
    attrib :block_id

    # @!attribute [r] block_height
    # Height of the block containing a transaction.
    # @return [Integer]
    attrib :block_height

    # @!attribute [r] position
    # Position of a transaction within the block.
    # @return [Integer]
    attrib :position

    # @!attribute [r] reference_data
    # User specified, unstructured data embedded within a transaction.
    # @return [Hash]
    attrib :reference_data

    # @!attribute [r] is_local
    # A flag indicating one or more inputs or outputs are local.
    # @return [Boolean]
    attrib :is_local

    # @!attribute [r] inputs
    # List of specified inputs for a transaction.
    # @return [Array<Input>]
    attrib(:inputs) { |raw| raw.map { |v| Input.new(v) } }

    # @!attribute [r] outputs
    # List of specified outputs for a transaction.
    # @return [Array<Output>]
    attrib(:outputs) { |raw| raw.map { |v| Output.new(v) } }

    class ClientModule < Chain::ClientModule
      # Build an unsigned transaction from a set of actions.
      # @param [Builder] builder Builder object with actions defined. If provided, overrides block parameter.
      # @yield Block defining transaction actions. A {Builder} object is passed as the only parameter.
      # @return [Template] Unsigned transaction template, or error.
      def build(builder = nil, &block)
        if builder.nil?
          builder = Builder.new(&block)
        end

        client.conn.singleton_batch_request(
          'build-transaction',
          [builder]
        ) { |item| Template.new(item) }
      end

      # Build multiple unsigned transactions from multiple sets of actions.
      # @param [Array<Builder>] builders Multiple builder objects with actions defined.
      # @return [BatchResponse<Template>] Batch of unsigned transaction templates, or errors.
      def build_batch(builders)
        client.conn.batch_request(
          'build-transaction',
          builders
        ) { |item| Template.new(item) }
      end

      # Submit a signed transaction to the blockchain.
      # @param [Template] template Signed transaction template.
      # @return [SubmitResponse]
      def submit(template)
        client.conn.singleton_batch_request(
          'submit-transaction',
          {transactions: [template]}
        ) { |item| SubmitResponse.new(item) }
      end

      # Submit multiple signed transactions to the blockchain.
      # @param [Array<Template>] templates Array of signed transaction templates.
      # @return [BatchResponse<SubmitResponse>]
      def submit_batch(templates)
        client.conn.batch_request(
          'submit-transaction',
          {transactions: templates}
        ) { |item| SubmitResponse.new(item) }
      end

      # List all transactions, optionally filtered
      # @param [Hash] opts Filtering information
      # @option opts [String] filter Filter string, see {https://chain.com/docs/core/build-applications/queries}.
      # @option opts [Array<String|Integer>] filter_params Parameter values for filter string (if needed).
      # @option opts [Integer] start_time A Unix timestamp in milliseconds. When specified, only transactions with a block time greater than the start time will be returned.
      # @option opts [Integer] end_time A Unix timestamp in milliseconds. When specified, only transactions with a block time less than the start time will be returned.
      # @option opts [Integer] timeout A time in milliseconds after which a server timeout should occur. Defaults to 1000 (1 second).
      # @return [Query]
      def query(opts = {})
        Query.new(client, opts)
      end
    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-transactions', query)
      end

      def translate(raw)
        Transaction.new(raw)
      end
    end

    class Input < ResponseObject
      # @!attribute [r] type
      # The type of the input.
      #
      # Possible values are "issue", "spend".
      # @return [String]
      attrib :type

      # @!attribute [r] asset_id
      # The id of the asset being issued or spent.
      # @return [String]
      attrib :asset_id

      # @!attribute [r] asset_alias
      # The alias of the asset being issued or spent (possibly null).
      # @return [String]
      attrib :asset_alias

      # @!attribute [r] asset_definition
      # The definition of the asset being issued or spent (possibly null).
      # @return [Hash]
      attrib :asset_definition

      # @!attribute [r] asset_tags
      # The tags of the asset being issued or spent (possibly null).
      # @return [Hash]
      attrib :asset_tags

      # @!attribute [r] asset_is_local
      # A flag indicating whether the asset being issued or spent is local.
      # @return [Boolean]
      attrib :asset_is_local

      # @!attribute [r] amount
      # The number of units of the asset being issued or spent.
      # @return [Integer]
      attrib :amount

      # @!attribute [r] spent_output_id
      # The id of the output consumed by this input. ID is nil if this is an issuance input.
      # @return [String]
      attrib :spent_output_id

      # @!attribute [r] account_id
      # The id of the account transferring the asset (possibly null if the
      # input is an issuance or an unspent output is specified).
      # @return [String]
      attrib :account_id

      # @!attribute [r] account_alias
      # The alias of the account transferring the asset (possibly null if the
      # input is an issuance or an unspent output is specified).
      # @return [String]
      attrib :account_alias

      # @!attribute [r] account_tags
      # The tags associated with the account (possibly null).
      # @return [String]
      attrib :account_tags

      # @!attribute [r] issuance_program
      # A program specifying a predicate for issuing an asset (possibly null
      # if input is not an issuance).
      # @return [String]
      attrib :issuance_program

      # @!attribute [r] reference_data
      # User specified, unstructured data embedded within an input
      # (possibly null).
      # @return [Hash]
      attrib :reference_data

      # @!attribute [r] is_local
      # A flag indicating if the input is local.
      # @return [Boolean]
      attrib :is_local
    end

    class Output < ResponseObject
      # @!attribute [r] id
      # The id of the output.
      # @return [String]
      attrib :id

      # @!attribute [r] type
      # The type of the output.
      #
      # Possible values are "control" and "retire".
      # @return [String]
      attrib :type

      # @!attribute [r] purpose
      # The purpose of the output.
      #
      # Possible values are "receive" and "change".
      # @return [String]
      attrib :purpose

      # @!attribute [r] position
      # The output's position in a transaction's list of outputs.
      # @return [Integer]
      attrib :position

      # @!attribute [r] asset_id
      # The id of the asset being controlled.
      # @return [String]
      attrib :asset_id

      # @!attribute [r] asset_alias
      # The alias of the asset being controlled (possibly null).
      # @return [String]
      attrib :asset_alias

      # @!attribute [r] asset_definition
      # The definition of the asset being controlled (possibly null).
      # @return [Hash]
      attrib :asset_definition

      # @!attribute [r] asset_tags
      # The tags of the asset being controlled (possibly null).
      # @return [Hash]
      attrib :asset_tags

      # @!attribute [r] asset_is_local
      # A flag indicating whether the asset being controlled is local.
      # @return [Boolean]
      attrib :asset_is_local

      # @!attribute [r] amount
      # The number of units of the asset being controlled.
      # @return [Integer]
      attrib :amount

      # @!attribute [r] account_id
      # The id of the account controlling this output (possibly null if a
      # control program is specified).
      # @return [String]
      attrib :account_id

      # @!attribute [r] account_alias
      # The alias of the account controlling this output (possibly null if
      # a control program is specified).
      # @return [String]
      attrib :account_alias

      # @!attribute [r] account_tags
      # The tags associated with the account controlling this output
      # (possibly null if a control program is specified).
      # @return [Hash]
      attrib :account_tags

      # @!attribute [r] control_program
      # The control program which must be satisfied to transfer this output.
      # @return [String]
      attrib :control_program

      # @!attribute [r] reference_data
      # User specified, unstructured data embedded within an input
      # (possibly null).
      # @return [Hash]
      attrib :reference_data

      # @!attribute [r] is_local
      # A flag indicating if the output is local.
      # @return [Boolean]
      attrib :is_local
    end

    class Builder
      def initialize(&block)
        block.call(self) if block
      end

      # @return [Array<Hash>]
      def actions
        @actions ||= []
      end

      # @param [Template, String] template_or_raw_tx
      # @return [Builder]
      def base_transaction(template_or_raw_tx)
        if template_or_raw_tx.is_a?(Transaction::Template)
          @base_transaction = template_or_raw_tx.raw_transaction
        else
          @base_transaction = template_or_raw_tx
        end
        self
      end

      # @return [Builder]
      def ttl(ttl)
        @ttl = ttl
        self
      end

      # @return [Hash]
      def to_h
        {
          actions: actions,
          base_transaction: @base_transaction,
          ttl: @ttl,
        }.select do |k,v|
          # TODO: Patches an issue in Chain Core 1.0 where nil values are rejected
          # Remove in 1.1.0 or later
          v != nil
        end
      end

      # @return [String]
      def to_json(opts = nil)
        to_h.to_json(opts)
      end

      # Add an action to the transaction builder
      # @param [Hash] params Action parameters containing a type field and the
      #               required parameters for that type
      # @return [Builder]
      def add_action(params)
        # Some actions require an idempotency token, so we'll add it here as a
        # generic parameter.
        params = {client_token: SecureRandom.uuid}.merge(params)
        actions << params
        self
      end

      # Sets the transaction-level reference data.
      # May only be used once per transaction.
      # @param [Hash] reference_data User specified, unstructured data to
      #                              be embedded in a transaction
      # @return [Builder]
      def transaction_reference_data(reference_data)
        add_action(
          type: :set_transaction_reference_data,
          reference_data: reference_data,
        )
      end

      # Add an issuance action.
      # @param [Hash] params Action parameters
      # @option params [String] :asset_id Asset ID specifying the asset to be issued.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :asset_alias Asset alias specifying the asset to be issued.
      #                                   You must specify either an ID or an alias.
      # @option params [Integer] :amount amount of the asset to be issued
      # @return [Builder]
      def issue(params)
        add_action(params.merge(type: :issue))
      end

      # Add a spend action taken on a particular account.
      # @param [Hash] params Action parameters
      # @option params [String] :asset_id Asset ID specifying the asset to be spent.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :asset_alias Asset alias specifying the asset to be spent.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :account_id Account ID specifying the account spending the asset.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :account_alias Account alias specifying the account spending the asset.
      #                                   You must specify either an ID or an alias.
      # @option params [Integer] :amount amount of the asset to be spent.
      # @return [Builder]
      def spend_from_account(params)
        add_action(params.merge(type: :spend_account))
      end

      # Add a spend action taken on a particular unspent output.
      # @param [Hash] params Action parameters
      # @option params [String] :output_id Output ID specifying the transaction output to spend.
      # @return [Builder]
      def spend_account_unspent_output(params)
        add_action(params.merge(type: :spend_account_unspent_output))
      end

      # Add a control action taken on a particular account.
      # @param [Hash] params Action parameters
      # @option params [String] :asset_id Asset ID specifying the asset to be controlled.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :asset_alias Asset alias specifying the asset to be controlled.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :account_id Account ID specifying the account controlling the asset.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :account_alias Account alias specifying the account controlling the asset.
      #                                   You must specify either an ID or an alias.
      # @option params [Integer] :amount amount of the asset to be controlled.
      # @return [Builder]
      def control_with_account(params)
        add_action(params.merge(type: :control_account))
      end

      # Sends assets to the specified receiver.
      #
      # @param [Hash] params Action parameters
      # @option params [Receiver] :control_program the receiver object.
      # @option params [String] :asset_id Asset ID specifying the asset to be controlled.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :asset_alias Asset alias specifying the asset to be controlled.
      #                                   You must specify either an ID or an alias.
      # @option params [Integer] :amount amount of the asset to be controlled.
      # @return [Builder]
      def control_with_receiver(params)
        add_action(params.merge(type: :control_receiver))
      end

      # Add a retire action.
      # @param [Hash] params Action parameters
      # @option params [String] :asset_id Asset ID specifying the asset to be retired.
      #                                   You must specify either an ID or an alias.
      # @option params [String] :asset_alias Asset alias specifying the asset to be retired.
      #                                   You must specify either an ID or an alias.
      # @option params [Integer] :amount Amount of the asset to be retired.
      # @return [Builder]
      def retire(params)
        add_action(params.merge(type: :retire))
      end
    end

    class SubmitResponse < ResponseObject
      # @!attribute [r] id
      # @return [String]
      attrib :id
    end

    class Template < ResponseObject
      # @!attribute [r] raw_transaction
      # @return [String]
      attrib :raw_transaction

      # @!attribute [r] signing_instructions
      # @return [String]
      attrib :signing_instructions

      # @return [Template]
      def allow_additional_actions
        @allow_additional_actions = true
        self
      end

      # @return [Hash]
      def to_h
        super.merge(allow_additional_actions: @allow_additional_actions)
      end
    end
  end
end
