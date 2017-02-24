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
      # @param [Hash] opts Filtering information
      # @option opts [String] filter Filter string, see {https://chain.com/docs/core/build-applications/queries}.
      # @option opts [Array<String|Integer>] filter_params Parameter values for filter string (if needed).
      # @option opts [Array<String>] sum_by List of unspent output attributes to sum by.
      # @option opts [Integer] timestamp A millisecond Unix timestamp. By using this parameter, you can perform queries that reflect the state of the blockchain at different points in time.
      # @return [Query]
      def query(opts = {})
        Query.new(client, opts)
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
