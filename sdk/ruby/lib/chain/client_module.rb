module Chain
  class ClientModule

    attr_reader :client

    def initialize(client)
      @client = client
    end

  end
end
