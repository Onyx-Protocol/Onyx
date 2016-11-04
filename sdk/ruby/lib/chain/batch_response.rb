def ensure_key_sorting(h)
  sorted = h.keys.sort
  return h if sorted == h.keys
  sorted.reduce({}) { |memo, k| memo[k] = h[k]; memo }
end

module Chain
  class BatchResponse
    def initialize(successes: {}, errors: {}, response: nil)
      @successes = ensure_key_sorting(successes)
      @errors = ensure_key_sorting(errors)
      @response = response
    end

    def size
      successes.size + errors.size
    end

    attr_reader :successes, :errors, :response
  end
end
