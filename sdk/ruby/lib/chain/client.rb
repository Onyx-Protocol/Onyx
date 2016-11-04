require_relative './access_token'
require_relative './account'
require_relative './asset'
require_relative './balance'
require_relative './config'
require_relative './constants'
require_relative './hsm_signer'
require_relative './mock_hsm'
require_relative './transaction'
require_relative './transaction_feed'
require_relative './unspent_output'

module Chain
  class Client

    def initialize(opts = {})
      @opts = {url: DEFAULT_API_HOST}.merge(opts)
    end

    def opts
      @opts.dup
    end

    # @return [Connection]
    def conn
      @conn ||= Connection.new(@opts)
    end

    # @return [AccessToken::ClientModule]
    def access_tokens
      @access_tokens ||= AccessToken::ClientModule.new(self)
    end

    # @return [Account::ClientModule]
    def accounts
      @accounts ||= Account::ClientModule.new(self)
    end

    # @return [Asset::ClientModule]
    def assets
      @assets ||= Asset::ClientModule.new(self)
    end

    # @return [Balance::ClientModule]
    def balances
      @balances ||= Balance::ClientModule.new(self)
    end

    # @return [Config::ClientModule]
    def config
      @config ||= Config::ClientModule.new(self)
    end

    # @return [MockHSM::ClientModule]
    def mock_hsm
      @mock_hsm ||= MockHSM::ClientModule.new(self)
    end

    # @return [Transaction::ClientModule]
    def transactions
      @transactions ||= Transaction::ClientModule.new(self)
    end

    # @return [TransactionFeed::ClientModule]
    def transaction_feeds
      @transaction_feeds ||= TransactionFeed::ClientModule.new(self)
    end

    # @return [UnspentOutput::ClientModule]
    def unspent_outputs
      @unspent_outputs ||= UnspentOutput::ClientModule.new(self)
    end

  end
end
