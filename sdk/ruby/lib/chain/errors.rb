module Chain

  # Base class for all errors raised by the Chain SDK.
  class BaseError < StandardError; end

  # InvalidRequestIDError arises when an HTTP response is received, but it does
  # not contain headers that are included in all Chain API responses. This
  # could arise due to a badly-configured proxy, or other upstream network
  # issues.
  class InvalidRequestIDError < BaseError
    attr_accessor :response

    def initialize(response)
      super "Response HTTP header field Chain-Request-ID is unset. There may be network issues. Please check your local network settings."
      self.response = response
    end
  end

  # JSONError should be very rare, and will only arise if there is a bug in the
  # Chain API, or if the upstream server is spoofing common Chain API response
  # headers.
  class JSONError < BaseError
    attr_accessor :request_id
    attr_accessor :response

    def initialize(request_id, response)
      super "Error decoding JSON response. Request-ID: #{request_id}"
      self.request_id = request_id
      self.response = response
    end
  end

  # APIError describes errors that are codified by the Chain API. They have
  # an error code, a message, and an optional detail field that provides
  # additional context for the error.
  class APIError < BaseError
    RETRIABLE_STATUS_CODES = [
      408, # Request Timeout
      429, # Too Many Requests
      500, # Internal Server Error
      502, # Bad Gateway
      503, # Service Unavailable
      504, # Gateway Timeout
      509, # Bandwidth Limit Exceeded
    ]

    attr_accessor :code, :chain_message, :detail, :data, :temporary, :request_id, :response

    def initialize(body, response)
      self.code = body['code']
      self.chain_message = body['message']
      self.detail = body['detail']
      self.temporary = body['temporary']

      self.response = response
      self.request_id = response['Chain-Request-ID'] if response

      super self.class.format_error_message(code, chain_message, detail, request_id)
    end

    def retriable?
      temporary || (response && RETRIABLE_STATUS_CODES.include?(Integer(response.code)))
    end

    def self.format_error_message(code, message, detail, request_id)
      tokens = []
      tokens << "Code: #{code}" if code.is_a?(String) && code.size > 0
      tokens << "Message: #{message}"
      tokens << "Detail: #{detail}" if detail.is_a?(String) && detail.size > 0
      tokens << "Request-ID: #{request_id}"
      tokens.join(' ')
    end
  end

  # UnauthorizedError is a special case of APIError, and is raised when the
  # response status code is 401. This is a common error case, so a discrete
  # exception type is provided for convenience.
  class UnauthorizedError < APIError; end

end
