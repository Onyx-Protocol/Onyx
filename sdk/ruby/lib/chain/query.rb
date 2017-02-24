module Chain
  class Query
    include ::Enumerable

    # @return [Client]
    attr_reader :client

    def initialize(client, first_query = {})
      @client = client
      @first_query = first_query
    end

    # Iterate through objects in response, fetching the next page of results
    # from the API as needed.
    #
    # Implements required method
    # {https://ruby-doc.org/core/Enumerable.html Enumerable#each}.
    # @return [void]
    def each
      page = fetch(@first_query)

      loop do
        if page['items'].empty? # we consume this array as we iterate
          break if page['last_page']
          page = fetch(page['next'])

          # The second predicate (empty?) *should* be redundant, but we check it
          # anyway as a defensive measure.
          break if page['items'].empty?
        end

        item = page['items'].shift
        yield translate(item)
      end
    end

    # @abstract
    def fetch(query)
      raise NotImplementedError
    end

    # Overwrite to translate API response data to a different Ruby object.
    # @abstract
    def translate(response_object)
      raise NotImplementedError
    end

    alias_method :all, :to_a
  end
end
