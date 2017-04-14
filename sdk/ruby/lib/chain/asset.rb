require 'securerandom'

require_relative './client_module'
require_relative './errors'
require_relative './query'
require_relative './response_object'

module Chain
  class Asset < ResponseObject

    # @!attribute [r] id
    # Globally unique identifier of the asset.
    # Asset version 1 specifies the asset id as the hash of:
    # - the asset version
    # - the asset's issuance program
    # - the core's VM version
    # - the hash of the network's initial block
    # @return [String]
    attrib :id

    # @!attribute [r] alias
    # User specified, unique identifier.
    # @return [String]
    attrib :alias

    # @!attribute [r] issuance_program
    # @return [String]
    attrib :issuance_program

    # @!attribute [r] keys
    # @return [Array<Key>]
    attrib(:keys) { |raw| raw.map { |v| Key.new(v) } }

    # @!attribute [r] quorum
    # @return [Integer]
    attrib :quorum

    # @!attribute [r] definition
    # User-specified, arbitrary/unstructured data visible across
    # blockchain networks. Version 1 assets specify the definition in their
    # issuance programs, rendering the definition immutable.
    # @return [Hash]
    attrib :definition

    # @!attribute [r] tags
    # @return [Hash]
    attrib :tags

    # @!attribute [r] is_local
    # @return [Boolean]
    attrib :is_local

    class ClientModule < Chain::ClientModule
      # @param [Hash] opts Options hash specifiying asset creation details.
      # @option opts [String] alias User specified, unique identifier.
      # @option opts [Array<String>] root_xpubs The list of keys used to create the issuance program for the asset.
      # @option opts [Integer] quorum The number of keys required to issue units of the asset.
      # @option opts [Hash] tags User-specified, arbitrary/unstructured data local to the asset's originating core.
      # @option opts [Hash] definition User-specified, arbitrary/unstructured data visible across blockchain networks.
      # @return [Asset]
      def create(opts)
        opts = {client_token: SecureRandom.uuid}.merge(opts)
        client.conn.singleton_batch_request('create-asset', [opts]) { |item| Asset.new(item) }
      end

      # @param [Array<Hash>] opts An array of options hashes. See {#create} for a description of the hash structure.
      # @return [BatchResponse<Asset>]
      def create_batch(opts)
        opts = opts.map { |i| {client_token: SecureRandom.uuid}.merge(i) }
        client.conn.batch_request('create-asset', opts) { |item| Asset.new(item) }
      end

      # @param [Hash] opts Options hash specifiying asset creation details.
      # @option opts [String] id The ID of the asset. Either an ID or alias should be provided, but not both.
      # @option opts [String] alias The alias of the asset. Either an ID or alias should be provided, but not both.
      # @option opts [Hash] tags A new set of tags, which will replace the existing tags.
      # @return [Hash] a success message.
      def update_tags(opts)
        client.conn.singleton_batch_request('update-asset-tags', [opts])
      end

      # @param [Array<Hash>] opts An array of options hashes. See {#update_tags} for a description of the hash structure.
      # @return [BatchResponse<Hash>]
      def update_tags_batch(opts)
        client.conn.batch_request('update-asset-tags', opts)
      end

      # @param [Hash] opts Filtering information
      # @option opts [String] filter Filter string, see {https://chain.com/docs/core/build-applications/queries}.
      # @option opts [Array<String|Integer>] filter_params Parameter values for filter string (if needed).
      # @return [Query]
      def query(opts = {})
        Query.new(client, opts)
      end
    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-assets', query)
      end

      def translate(raw)
        Asset.new(raw)
      end
    end

    class Key < ResponseObject
      # @!attribute [r] root_xpub
      # Hex-encoded representation of the root extended public key.
      # @return [String]
      attrib :root_xpub

      # @!attribute [r] asset_pubkey
      # The derived public key, used in the asset's issuance program.
      # @return [String]
      attrib :asset_pubkey

      # @!attribute [r] asset_derivation_path
      # The derivation path of the extended key.
      # @return [String]
      attrib :asset_derivation_path
    end

  end
end
