require_relative './client_module'
require_relative './control_program'
require_relative './errors'
require_relative './query'
require_relative './receiver'
require_relative './response_object'

module Chain
  class Account < ResponseObject

    # @!attribute [r] id
    # Unique account identifier.
    # @return [String]
    attrib :id

    # @!attribute [r] alias
    # User specified, unique identifier.
    # @return [String]
    attrib :alias

    # @!attribute [r] keys
    # The list of keys used to create control programs under the account.
    # Signatures from these keys are required for spending funds held in the account.
    # @return [Array<Key>]
    attrib(:keys) { |raw| raw.map { |v| Key.new(v) } }

    # @!attribute [r] quorum
    # The number of keys required to sign transactions for the account.
    # @return [Integer]
    attrib :quorum

    # @!attribute [r] tags
    # User-specified tag structure for the account.
    # @return [Hash]
    attrib :tags

    class ClientModule < Chain::ClientModule
      # @param [Hash] opts
      # @return [Account]
      def create(opts)
        opts = {client_token: SecureRandom.uuid}.merge(opts)
        client.conn.singleton_batch_request('create-account', [opts]) { |item| Account.new(item) }
      end

      # @param [Array<Hash>] opts
      # @return [Array<Account>]
      def create_batch(opts)
        opts = opts.map { |i| {client_token: SecureRandom.uuid}.merge(i) }
        client.conn.batch_request('create-account', opts) { |item| Account.new(item) }
      end

      # @deprecated (as of version 1.1) Use {#create_receiver} instead.
      # @param [Hash] opts
      # @return [ControlProgram]
      def create_control_program(opts = {})
        # We don't use keyword params here because 'alias' is a Ruby reserverd
        # word.
        params = {}
        params[:account_alias] = opts[:alias] if opts.key?(:alias)
        params[:account_id] = opts[:id] if opts.key?(:id)

        client.conn.singleton_batch_request(
          'create-control-program',
          [{type: :account, params: params}]
        ) { |item| ControlProgram.new(item) }
      end

      # Creates a new receiver under the specified account.
      #
      # @param opts [Hash] Options hash
      # @option opts [String] :account_alias Unique alias for an account. Either account_alias or account_id is required.
      # @option opts [String] :account_id Unique ID for an account. Either account_alias or account_id is required.
      # @option opts [String] :expires_at An RFC3339 timestamp indicating when the receiver will expire. Defaults to 30 days in the future.
      # @return [Receiver]
      def create_receiver(opts)
        client.conn.singleton_batch_request('create-account-receiver', [opts]) { |item| Receiver.new(item) }
      end

      # Creates new receivers under the specified accounts.
      #
      # @param opts_list [Array<Hash>] Array of options hashes. See {#create_receiver} for a description of the hash structure.
      # @return [BatchResponse]
      def create_receiver_batch(opts_list)
        client.conn.batch_request('create-account-receiver', opts_list) { |item| Receiver.new(item) }
      end

      # @param [Hash] query
      # @return [Query]
      def query(query = {})
        Query.new(client, query)
      end
    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-accounts', query)
      end

      def translate(raw)
        Account.new(raw)
      end
    end

    class Key < ResponseObject
      # @!attribute [r] root_xpub
      # Hex-encoded representation of the root extended public key.
      # @return [String]
      attrib :root_xpub

      # @!attribute [r] account_xpub
      # The extended public key used to create control programs for the account.
      # @return [String]
      attrib :account_xpub

      # @!attribute [r] account_derivation_path
      # The derivation path of the extended key.
      # @return [String]
      attrib :account_derivation_path
    end

  end
end
