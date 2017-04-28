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

    # @deprecated
    # @!attribute [r] type
    # Either 'client' or 'network'. Ignore in 1.2 or greater.
    # @return [String]
    attrib :type

    # @!attribute [r] created_at
    # Timestamp of token creation.
    # @return [Time]
    attrib :created_at, rfc3339_time: true

    class ClientModule < Chain::ClientModule

      # Create an access token.
      # @param [Hash] opts
      # @option opts [String] :id ID specifying the ID of newly created access token.
      #                                   You must specify a unique ID for access token.
      # @option opts [String] :type DEPRECATED. Do not use in 1.2 or greater.
      # @return [AccessToken]
      def create(opts = {})
        AccessToken.new(client.conn.request('create-access-token', opts))
      end

      # Get all access tokens sorted by descending creation time,
      # optionally filtered by type.
      # @param [Hash] opts Filtering information
      # @option opts [String] :type DEPRECATED. Do not use in 1.2 or greater.
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
        nil
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
