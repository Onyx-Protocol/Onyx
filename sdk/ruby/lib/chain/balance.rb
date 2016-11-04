require_relative './client_module'
require_relative './response_object'
require_relative './query'

module Chain
  class Balance < ResponseObject

    # @!attribute [r] amount
    # Sum of the unspent outputs.
    # @return [Integer]
    attrib :amount

    # @!attribute [r] sum_by
    # List of parameters on which to sum unspent outputs.
    # @return [Hash<String => String>]
    attrib :sum_by

    class ClientModule < Chain::ClientModule
      # @param [Hash] query
      # @return [Query]
      def query(query = {})
        Query.new(client, query)
      end
    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-balances', query)
      end

      def translate(raw)
        Balance.new(raw)
      end
    end
  end
end
