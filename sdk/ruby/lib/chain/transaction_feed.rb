require 'securerandom'

require_relative './client_module'
require_relative './connection'
require_relative './constants'
require_relative './query'
require_relative './response_object'

module Chain
  class TransactionFeed < ResponseObject

    # @!attribute [r] id
    # Unique transaction feed identifier.
    # @return [String]
    attrib :id

    # @!attribute [r] alias
    # User specified, unique identifier.
    # @return [String]
    attrib :alias

    # @!attribute [r] filter
    # @return [String]
    attrib :filter

    # @!attribute [r] after
    # @return [String]
    attrib :after

    def initialize(raw_attribs, base_conn)
      super(raw_attribs)

      # The consume/ack cycle should run on its own thread, so make a copy of
      # the base connection so this feed has an exclusive HTTP connection.
      @conn = Connection.new(base_conn.opts)
    end

    # @param [Fixnum] timeout value in seconds
    # @yield [Transaction] block process individual transactions
    # @yieldparam [Transaction] tx
    # @return [void]
    def consume(timeout: 24*60*60)
      query = {
        filter: filter,
        after: after,
        timeout: (timeout * 1000).to_i, # milliseconds
        ascending_with_long_poll: true
      }

      longpoll = Connection.new(@conn.opts.merge(read_timeout: timeout))

      loop do
        page = longpoll.request('list-transactions', query)
        query = page['next']

        page['items'].each do |raw_tx|
          tx = Transaction.new(raw_tx)

          # Memoize the cursor value for this transaction in case the user
          # decides to ack. The format of the cursor value is specified in the
          # core/query package.
          @next_after = "#{tx.block_height}:#{tx.position}-#{MAX_BLOCK_HEIGHT}"

          yield tx
        end
      end
    end

    def ack
      raise 'ack must be called at most once per cycle in a consume loop' unless @next_after

      @conn.request(
        'update-transaction-feed',
        id: id,
        after: @next_after,
        previous_after: after,
      )

      self.after = @next_after
      @next_after = nil
    end

    class ClientModule < Chain::ClientModule

      # @param [Hash] opts
      # @return [TransactionFeed]
      def create(opts)
        opts = {client_token: SecureRandom.uuid()}.merge(opts)
        TransactionFeed.new(client.conn.request('create-transaction-feed', opts), client.conn)
      end

      # @param [Hash] opts
      # @return [TransactionFeed]
      def get(opts)
        TransactionFeed.new(client.conn.request('get-transaction-feed', opts), client.conn)
      end

      # @param [Hash] opts
      # @option opts [String] :id ID of the transaction feed. You must provide either :id or :alias.
      # @option opts [String] :alias ID of the transaction feed. You must provide either :id or :alias.
      # @return [void]
      def delete(opts)
        client.conn.request('delete-transaction-feed', opts)
        nil
      end

      # @return [Query]
      def query
        Query.new(client)
      end

    end

    class Query < Chain::Query
      def fetch(query)
        client.conn.request('list-transaction-feeds', query)
      end

      def translate(raw)
        TransactionFeed.new(raw, client.conn)
      end
    end

  end
end
