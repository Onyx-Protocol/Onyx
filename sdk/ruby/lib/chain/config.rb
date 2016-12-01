require_relative './client_module'
require_relative './response_object'

module Chain
  module Config

    class Info < ResponseObject
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
      attrib(:configured_at) { |raw| Time.parse(raw) }

      # @!attribute [r] is_signer
      # @return [Boolean]
      attrib :is_signer

      # @!attribute [r] is_generator
      # @return [Boolean]
      attrib :is_generator

      # @!attribute [r] is_generator
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
      attrib(:generator_block_height_fetched_at) { |raw| Time.parse(raw) }

      # @!attribute [r] is_production
      # @return [Boolean]
      attrib :is_production

      # @!attribute [r] network_rpc_version
      # @return [Integer]
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

      # @!attribute [r] health
      # @return [Hash]
      attrib :health

      # @!attribute [r] snapshot
      # @return [Snapshot]
      attrib(:snapshot) { |raw| Snapshot.new(raw) }
    end

    class ClientModule < Chain::ClientModule
      # @return [void]
      def reset(everything: false)
        client.conn.request('reset', {everything: everything})
      end

      # @return [void]
      def configure(opts)
        client.conn.request('configure', opts)
      end

      # @return [Info]
      def info
        Info.new(client.conn.request('info'))
      end
    end

  end
end
