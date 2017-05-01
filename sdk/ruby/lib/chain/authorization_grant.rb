require_relative './client_module'
require_relative './query'
require_relative './response_object'

module Chain
  class AuthorizationGrant < ResponseObject

    # @!attribute [r] guard_type
    # The type of credential that the guard matches against. Only "access_token"
    # and "x509" are allowed.
    # @return [String]
    attrib :guard_type

    # @!attribute [r] guard_data
    # A list of parameters that match specific credentials.
    # @return [Hash]
    attrib :guard_data

    # @!attribute [r] policy
    # @return [String]
    attrib :policy

    # @!attribute [r] protected
    # @return [Boolean]
    # Whether the grant can be deleted. Only used for internal purposes.
    attrib :protected

    # @!attribute [r] created_at
    # Timestamp of token creation.
    # @return [Time]
    attrib :created_at, rfc3339_time: true

    class ClientModule < Chain::ClientModule

      # Create an authorization grant, which provides the specified
      # credential with access to the given policy. Credentials are identified
      # using predicates called guards. Guards identify credentials by type
      # and by patterns specific to that type.
      #
      # @param [Hash] opts
      # @option opts [String] :guard_type Either "access_token" or "x509".
      # @option opts [Hash] :guard_data Parameters that describe a credential.
      #
      #   For guards of type "access_token", provide a Hash with a single key,
      #   "id", whose value is the unique ID of the access token.
      #
      #   For guards of type "x509", there should be a single top-level key,
      #   "subject", which maps to a hash of Subject attributes. Valid
      #   keys include:
      #     - "C" (Country, string or array of strings)
      #     - "O" (Organization, string or array of strings)
      #     - "OU" (Organizational Unit, string or array of strings)
      #     - "L" (Locality, string or array of strings)
      #     - "ST" (State or Province, string or array of strings)
      #     - "STREET" (Street Address, string or array of strings)
      #     - "POSTALCODE" (Postal Code, string or array of strings)
      #     - "SERIALNUMBER" (Serial Number, string)
      #     - "CN" (Common Name, string)
      #
      # @option opts [String] :policy One of the following:
      #
      #   - "client-readwrite": full access to the Client API, including
      #      accounts, assets, transactions, access tokens, MockHSM, etc.
      #   - "client-readonly": read-only access to the Client API.
      #   - "monitoring": read-only access to diagnostic components of the API,
      #      including fetching configuration info.
      #   - "crosscore": access to the cross-core API, including fetching blocks
      #      and submitting transactions, but not including block signing.
      #   - "crosscore-signblock": access to the cross-core API's block-signing
      #      API call.
      # @return [AuthorizationGrant]
      def create(opts)
        # Copy input and stringify keys
        opts = opts.reduce({}) do |memo, (k, v)|
          memo[k.to_s] = v
          memo
        end

        if opts['guard_type'].to_s == 'x509'
          opts['guard_data'] = self.class.sanitize_x509(opts['guard_data'])
        end

        AuthorizationGrant.new(client.conn.request('create-authorization-grant', opts))
      end

      # List all authorization grants. The sort order is not defined.
      # @return [Array<AuthorizationGrant>]
      def list_all
        client.conn.request('list-authorization-grants')['items'].map { |item| AuthorizationGrant.new(item) }
      end

      # Delete the specified authorization grant.
      # @param opts Identical to {#create}.
      # @return [void]
      def delete(opts)
        client.conn.request('delete-authorization-grant', opts)
        nil
      end

      SUBJECT_ATTRIBUTES = {
        'C' => {array: true},
        'O' => {array: true},
        'OU' => {array: true},
        'L' => {array: true},
        'ST' => {array: true},
        'STREET' => {array: true},
        'POSTALCODE' => {array: true},
        'SERIALNUMBER' => {array: false},
        'CN' => {array: false},
      }

      def self.sanitize_x509(guard_data)
        first_key = guard_data.keys.first
        if guard_data.size != 1 || first_key.to_s.downcase != 'subject'
          raise ArgumentError.new('Guard data must contain exactly one key, "subject"')
        end

        res = {}
        res[first_key] = guard_data.values.first.reduce({}) do |memo, (k, v)|
          attrib = SUBJECT_ATTRIBUTES[k.to_s.upcase]
          raise ArgumentError.new("Invalid subject attrib: #{k}") unless attrib

          if attrib[:array] && !v.is_a?(Array)
            memo[k] = [v]
          elsif !attrib[:array] && v.is_a?(Array)
            raise ArgumentError.new("Invalid array value for #{k}: #{v}")
          else
            memo[k] = v
          end

          memo
        end
        res
      end

    end

  end
end
