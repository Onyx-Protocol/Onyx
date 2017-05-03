require_relative './client_module'
require_relative './response_object'

module Chain
  module Config

    class Info < ResponseObject
      class BuildConfig < ResponseObject
        # @!attribute [r] is_localhost_auth
        # @return [Boolean]
        # Whether any request from the loopback device (localhost) should be
        # automatically authenticated and authorized, without additional
        # credentials.
        attrib :is_localhost_auth

        # @!attribute [r] is_mockhsm
        # @return [Boolean]
        # Whether the MockHSM API is enabled.
        attrib :is_mockhsm

        # @!attribute [r] is_reset
        # @return [Boolean]
        # Whether the core reset API call is enabled.
        attrib :is_reset

        # @!attribute [r] is_http_ok
        # @return [Boolean]
        # Whether non-TLS HTTP requests (http://...) are allowed.
        attrib :is_http_ok
      end

      class Snapshot < ResponseObject
        # @!attribute [r] attempt
        # @return [Integer]
        attrib :attempt

        # @!attribute [r] height
        # @return [Integer]
        attrib :height

        # @!attribute [r] size
        # @return [Integer]
        attrib :size

        # @!attribute [r] downloaded
        # @return [Integer]
        attrib :downloaded

        # @!attribute [r] in_progress
        # @return [Boolean]
        attrib :in_progress
      end

      # @!attribute [r] is_configured
      # @return [Boolean]
      attrib :is_configured

      # @!attribute [r] configured_at
      # @return [Time]
      attrib :configured_at, rfc3339_time: true

      # @!attribute [r] is_signer
      # @return [Boolean]
      attrib :is_signer

      # @!attribute [r] is_generator
      # @return [Boolean]
      attrib :is_generator

      # @!attribute [r] generator_url
      # @return [String]
      attrib :generator_url

      # @!attribute [r] generator_access_token
      # @return [String]
      attrib :generator_access_token

      # @!attribute [r] blockchain_id
      # @return [String]
      attrib :blockchain_id

      # @!attribute [r] block_height
      # @return [Integer]
      attrib :block_height

      # @!attribute [r] generator_block_height
      # @return [Integer]
      attrib :generator_block_height

      # @!attribute [r] generator_block_height_fetched_at
      # @return [Time]
      attrib :generator_block_height_fetched_at, rfc3339_time: true

      # @!attribute [r] is_production
      # @return [Boolean]
      attrib :is_production

      # @!attribute [r] crosscore_rpc_version
      # @return [Integer]
      attrib :crosscore_rpc_version

      # @deprecated
      # @!attribute [r] network_rpc_version
      # @return [Integer]
      # Ignore in 1.2 or greater. Superseded by crosscore_rpc_version.
      attrib :network_rpc_version

      # @!attribute [r] core_id
      # @return [String]
      attrib :core_id

      # @!attribute [r] version
      # @return [String]
      attrib :version

      # @!attribute [r] build_commit
      # @return [String]
      attrib :build_commit

      # @!attribute [r] build_date
      # Date when the core binary was compiled.
      #
      # The API may not return this field as an RFC3399 timestamp,
      # so it is not converted into a Time object.
      # @return [String]
      attrib :build_date

      # @!attribute [r] build_config
      # @return [BuildConfig]
      attrib(:build_config) { |raw| BuildConfig.new(raw) }

      # @!attribute [r] health
      # @return [Hash]
      attrib :health

      # @!attribute [r] snapshot
      # @return [Snapshot]
      attrib(:snapshot) { |raw| Snapshot.new(raw) }
    end

    class ClientModule < Chain::ClientModule
      # Reset specified Chain Core.
      # @param [Boolean] everything 	If `true`, all objects including access tokens and MockHSM keys will be deleted. If `false`, then access tokens and MockHSM keys will be preserved.
      # @return [void]
      def reset(everything: false)
        client.conn.request('reset', {everything: everything})
      end

      # Configure specified Chain Core.
      # @param [Hash] opts Options for configuring Chain Core.
      # @option opts [Boolean] is_generator Whether the local core will be a block generator for the blockchain; i.e., you are starting a new blockchain on the local core. `false` if you are connecting to a pre-existing blockchain.
      # @option opts [String] generator_url A URL for the block generator. Required if `isGenerator` is false.
      # @option opts [String] generator_access_token 	An access token provided by administrators of the block generator. Required if `isGenerator` is false.
      # @option opts [String] blockchain_id The unique ID of the generator's blockchain. Required if `isGenerator` is false.
      # @return [void]
      def configure(opts)
        client.conn.request('configure', opts)
      end

      # Get info on specified Chain Core.
      # @return [Info]
      def info
        Info.new(client.conn.request('info'))
      end
    end

  end
end
