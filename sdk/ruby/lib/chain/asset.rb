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
      # @param [Hash] opts
      # @return [Asset]
      def create(opts)
        opts = {client_token: SecureRandom.uuid}.merge(opts)
        client.conn.singleton_batch_request('create-asset', [opts]) { |item| Asset.new(item) }
      end

      # @param [Hash] opts
      # @return [Array<Asset>]
      def create_batch(opts)
        opts = opts.map { |i| {client_token: SecureRandom.uuid}.merge(i) }
        client.conn.batch_request('create-asset', opts) { |item| Asset.new(item) }
      end

      # @param [Hash] query
      # @return [Query]
      def query(query = {})
        Query.new(client, query)
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
