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

      # Create client/network access token.
      # @param [Hash] opts
      # @option params [String] :type Type specifiying the type of access token to be created.
      #                                   You must specify either 'client' or 'network'.
      # @option params [String] :id ID specifying the ID of newly created access token.
      #                                   You must specify a unique ID for access token.
      # @return [AccessToken]
      def create(opts = {})
        AccessToken.new(client.conn.request('create-access-token', opts))
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
