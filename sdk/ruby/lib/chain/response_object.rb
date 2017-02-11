require 'json'
require 'time'

module Chain
  class ResponseObject
    def initialize(raw_attribs)
      raw_attribs.each do |k, v|
        next unless self.class.has_attrib?(k)
        self[k] = self.class.translate(k, v) unless v.nil?
      end
    end

    def to_h
      self.class.attrib_opts.keys.reduce({}) do |memo, name|
        memo[name] = instance_variable_get("@#{name}")
        memo
      end
    end

    def to_json(opts = nil)
      h = to_h.reduce({}) do |memo, (k, v)|
        memo[k] = self.class.detranslate(k, v)
        memo
      end

      h.to_json
    end

    def [](attrib_name)
      attrib_name = attrib_name.to_sym
      raise KeyError.new("key not found: #{attrib_name}") unless self.class.attrib_opts.key?(attrib_name)

      instance_variable_get "@#{attrib_name}"
    end

    def []=(attrib_name, value)
      attrib_name = attrib_name.to_sym
      raise KeyError.new("key not found: #{attrib_name}") unless self.class.attrib_opts.key?(attrib_name)

      instance_variable_set "@#{attrib_name}", value
    end

    # @!visibility private
    def self.attrib_opts
      @attrib_opts ||= {}
    end

    # @!visibility private
    def self.attrib(attrib_name, opts = {}, &translate)
      opts[:translate] = translate
      attrib_opts[attrib_name.to_sym] = opts
      attr_accessor attrib_name
    end

    # @!visibility private
    def self.has_attrib?(attrib_name)
      attrib_opts.key?(attrib_name.to_sym)
    end

    # @!visibility private
    def self.translate(attrib_name, raw_value)
      attrib_name = attrib_name.to_sym
      opts = attrib_opts[attrib_name]

      return Time.parse(raw_value) if opts[:rfc3339_time]
      return raw_value if opts[:translate].nil?

      begin
        opts[:translate].call raw_value
      rescue => e
        raise TranslateError.new(attrib_name, raw_value, e)
      end
    end

    # @!visibility private
    def self.detranslate(attrib_name, raw_value)
      opts = attrib_opts.fetch(attrib_name, {})

      if opts[:rfc3339_time]
        begin
          return raw_value.to_datetime.rfc3339
        rescue => e
          raise DetranslateError.new(attrib_name, raw_value, e)
        end
      end

      raw_value
    end

    class TranslateError < StandardError
      attr_reader :attrib_name
      attr_reader :raw_value
      attr_reader :source

      def initialize(attrib_name, raw_value, source)
        super "Error translating attrib #{attrib_name}: #{source}"
        @attrib_name = attrib_name
        @raw_value = raw_value
        @source = source
      end
    end

    class DetranslateError < StandardError
      attr_reader :attrib_name
      attr_reader :raw_value
      attr_reader :source

      def initialize(attrib_name, raw_value, source)
        super "Error de-translating attrib #{attrib_name}: #{source}"
        @attrib_name = attrib_name
        @raw_value = raw_value
        @source = source
      end
    end
  end
end
