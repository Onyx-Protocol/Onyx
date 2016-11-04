require_relative './client_module'
require_relative './response_object'
require_relative './query'

module Chain
  class UnspentOutput < ResponseObject

    # @!attribute [r] type
    # @return [String]
    attrib :type

    # @!attribute [r] purpose
    # @return [String]
    attrib :purpose

    # @!attribute [r] transaction_id
    # @return [String]
    attrib :transaction_id

    # @!attribute [r] position
    # @return [Integer]
    attrib :position

    # @!attribute [r] asset_id
    # @return [String]
    attrib :asset_id

    # @!attribute [r] asset_alias
    # @return [String]
    attrib :asset_alias

    # @!attribute [r] asset_definition
    # @return [Hash]
    attrib :asset_definition

    # @!attribute [r] asset_tags
    # @return [Hash]
    attrib :asset_tags

    # @!attribute [r] asset_is_local
    # @return [Boolean]
    attrib :asset_is_local

    # @!attribute [r] amount
    # @return [Integer]
    attrib :amount

    # @!attribute [r] account_id
    # @return [String]
    attrib :account_id

    # @!attribute [r] account_alias
    # @return [String]
    attrib :account_alias

    # @!attribute [r] account_tags
    # @return [Hash]
    attrib :account_tags

    # @!attribute [r] control_program
    # @return [String]
    attrib :control_program

    # @!attribute [r] reference_data
    # @return [Hash]
    attrib :reference_data

    # @!attribute [r] is_local
    # @return [Boolean]
    attrib :is_local

    class ClientModule < Chain::ClientModule
      # @param [Hash] query
      # @return Query
      def query(query = {})
        Query.new(client, query)
      end
    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-unspent-outputs', query)
      end

      def translate(raw)
        UnspentOutput.new(raw)
      end
    end
  end
end
