require_relative './client_module'
require_relative './query'
require_relative './response_object'

module Chain
  class AccessToken < ResponseObject

    # @!attribute [r] id
    # User specified, unique identifier.
    # @return [String]
    attrib :id

    # @!attribute [r] token
    # Only returned in the response from {ClientModule.create}.
    # @return [String]
    attrib :token

    # @!attribute [r] type
    # Either 'client' or 'network'.
    # @return [String]
    attrib :type

    # @!attribute [r] created_at
    # Timestamp of token creation.
    # @return [Time]
    attrib(:created_at) { |raw| Time.parse(raw) }

    class ClientModule < Chain::ClientModule

      # @return [AccessToken]
      def create(type:, id:)
        AccessToken.new(client.conn.request(
          'create-access-token',
          {type: type, id: id}
        ))
      end

      # @param [Hash] opts
      # @return [Query]
      def query(opts = {})
        Query.new(client, opts)
      end

      # Delete the access token specified.
      # @param [String] id access token ID
      # @raise [APIError]
      # @return [void]
      def delete(id)
        client.conn.request('delete-access-token', {id: id})
        return
      end

      class Query < Chain::Query
        def fetch(query)
          client.conn.request('list-access-tokens', query)
        end

        def translate(raw)
          AccessToken.new(raw)
        end
      end

    end

  end
end
