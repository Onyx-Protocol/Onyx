require_relative './response_object'

module Chain
  class Receiver < ResponseObject
    # @!attribute [r] control_program
    # The underlying control program that will be used in transactions paying to this receiver.
    # @return [String]
    attrib :control_program

    # @!attribute [r] expires_at
    # A timestamp indicating when the receiver will expire.
    # @return [Time]
    attrib :expires_at, rfc3339_time: true
  end
end
