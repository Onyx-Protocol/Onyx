require_relative './client_module'
require_relative './connection'
require_relative './query'
require_relative './response_object'

module Chain
  class MockHSM

    class ClientModule < Chain::ClientModule
      # @return [Key::ClientModule]
      def keys
        @keys_module ||= Key::ClientModule.new(client)
      end

      # @return [Connection]
      def signer_conn
        return @signer_conn if @signer_conn

        opts = client.conn.opts
        opts[:url] += '/mockhsm'

        @signer_conn = Connection.new(opts)
      end
    end

    class Key < ResponseObject
      # @!attribute [r] alias
      # User specified, unique identifier of the key.
      # @return [String]
      attrib :alias

      # @!attribute [r] xpub
      # Hex-encoded string representation of the key.
      # @return [String]
      attrib :xpub

      class ClientModule < Chain::ClientModule

        # Creates a key object.
        # @param [Hash] opts
        # @return [Key]
        def create(opts = {})
          Key.new(client.conn.request('mockhsm/create-key', opts))
        end

        # @param [Hash] query
        # @return [Query]
        def query(query = {})
          Query.new(client, query)
        end
      end

      class Query < Chain::Query
        def fetch(query)
          client.conn.request('mockhsm/list-keys', query)
        end

        def translate(obj)
          Key.new(obj)
        end
      end
    end

  end
end
