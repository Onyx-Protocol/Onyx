require_relative './response_object'

module Chain
  class ControlProgram < ResponseObject
    # @!attribute [r] control_program
    # Hex-encoded string representation of the control program.
    # @return [String]
    attrib :control_program
  end
end
