require 'json'
require 'net/http'
require 'net/https'
require 'openssl'
require 'thread'

require_relative './batch_response'
require_relative './errors'
require_relative './version'

module Chain
  class Connection

    # Parameters to the retry exponential backoff function.
    MAX_RETRIES = 10
    RETRY_BASE_DELAY_MS = 40
    RETRY_MAX_DELAY_MS = 4000

    NETWORK_ERRORS = [
      InvalidRequestIDError,
      SocketError,
      EOFError,
      IOError,
      Timeout::Error,
      Errno::ECONNABORTED,
      Errno::ECONNRESET,
      Errno::ETIMEDOUT,
      Errno::EHOSTUNREACH,
      Errno::ECONNREFUSED,
    ]

    def initialize(opts)
      @opts = opts
      @url = URI(@opts[:url])
      @access_token = @opts[:access_token] || @url.userinfo
      @http_mutex = Mutex.new
    end

    # Returns a copy of the configuration options
    def opts
      @opts.dup
    end

    def request(path, body = {})
      _request_with_retries(path, body)[:body]
    end

    def batch_request(path, body = {}, &translate)
      res = _request_with_retries(path, body)
      body = res[:body]
      response = res[:response]

      successes = {}
      errors = {}

      body.each_with_index do |item, i|
        if !!item['code']
          errors[i] = APIError.new(item, response)
        else
          successes[i] = translate.call(item)
        end
      end

      BatchResponse.new(
        successes: successes,
        errors: errors,
        response: response,
      )
    end

    def singleton_batch_request(path, body = {}, &translate)
      batch = batch_request(path, body, &translate)

      if batch.size != 1
        raise "Invalid response, expected a single response object but got #{batch.items.size}"
      end

      raise batch.errors.values.first if batch.errors.size == 1

      batch.successes.values.first
    end

    private

    def _request_with_retries(path, body)
      attempts = 0

      begin
        attempts += 1

        # If this is a retry and not the first attempt, sleep before making the
        # retry request.
        sleep(backoff_delay(attempts)) if attempts > 1

        _single_request(path, body)
      rescue *NETWORK_ERRORS => e
        raise e if attempts > MAX_RETRIES
        retry
      rescue APIError => e
        raise e if attempts > MAX_RETRIES
        retry if e.retriable?
        raise e
      end
    end

    def _single_request(path, body)
      @http_mutex.synchronize do
        # Timeout configuration
        [:open_timeout, :read_timeout, :ssl_timeout].each do |k|
          next unless @opts.key?(k)
          http.send "#{k}=", @opts[k]
        end

        req = Net::HTTP::Post.new(@url.request_uri + path)
        req['Accept'] = 'application/json'
        req['Content-Type'] = 'application/json'
        req['User-Agent'] = 'chain-sdk-ruby/' + Chain::VERSION
        req.body = JSON.dump(body)

        if @access_token
          user, pass = @access_token.split(':')
          req.basic_auth(user, pass)
        end

        response = http.request(req)

        req_id = response['Chain-Request-ID']
        unless req_id.is_a?(String) && req_id.size > 0
          raise InvalidRequestIDError.new(response)
        end

        status = Integer(response.code)
        parsed_body = nil

        if status != 204 # No Content
          begin
            parsed_body = JSON.parse(response.body)
          rescue JSON::JSONError
            raise JSONError.new(req_id, response)
          end
        end

        if status / 100 != 2
          klass = status == 401 ? UnauthorizedError : APIError
          raise klass.new(parsed_body, response)
        end

        {body: parsed_body, response: response}
      end
    end

    private

    MILLIS_TO_SEC = 0.001

    def backoff_delay(attempt)
      max = RETRY_BASE_DELAY_MS * 2**(attempt-1)
      max = [max, RETRY_MAX_DELAY_MS].min
      millis = rand(max) + 1
      millis * MILLIS_TO_SEC
    end

    def http
      return @http if @http

      args = [@url.host, @url.port]

      # Proxy configuration
      if @opts.key?(:proxy_addr)
        args += [@opts[:proxy_addr], @opts[:proxy_port]]
        if @opts.key?(:proxy_user)
          args += [@opts[:proxy_user], @opts[:proxy_pass]]
        end
      end

      @http = Net::HTTP.new(*args)

      @http.set_debug_output($stdout) if ENV['DEBUG']
      if @url.scheme == 'https'
        @http.use_ssl = true
        @http.verify_mode = OpenSSL::SSL::VERIFY_PEER
      end

      @http
    end
  end
end
