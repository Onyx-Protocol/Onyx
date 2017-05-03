const shared = require('../shared')
const util = require('../util')

/**
 * Authorization grants provide a mapping from guard objects (access tokens or X509
 * certificates) to a list of predefined Chain Core access policies.
 *
 * * **client-readwrite**: full access to the Client API
 * * **client-readonly**: access to read-only Client endpoints
 * * **monitoring**: access to monitoring-specific endpoints
 * * **crosscore**: access to the cross-core API, including fetching blocks and
 *   submitting transactions to the generator, but not including block signing
 * * **crosscore-signblock**: access to the cross-core API's block singing
 *   functionality
 *
 * More info: {@link https://chain.com/docs/core/learn-more/authentication-and-authorization}
 * @typedef {Object} AuthorizationGrant
 * @global
 *
 * @property {String} guardType
 * Type of credential, either 'access_token' or 'x509'.
 *
 * @property {Object} guardData
 * Data used by the guard to identity incoming credentials.
 *
 * If guardType is 'access_token', you should provide an instance of
 * {@link module:AuthorizationGrantsApi~AccessTokenGuardData}, which identifies access tokens by ID.
 *
 * If guardType is 'x509', you should provide an instance of {@link module:AuthorizationGrantsApi~X509GuardData},
 * which identifies x509 certificates based on kev-value pairs in specified
 * certificate fields.
 *
 * @property {String} policy
 * Authorization single policy to attach to specific grant.
 *
 * @property {Boolean} protected
 * Whether the grant can be deleted. Only used for internal purposes.
 *
 * @property {String} createdAt
 * Time of grant creation, RFC3339 formatted.
 */

/**
 * API for interacting with {@link AuthorizationGrant access grants}.
 *
 * More info: {@link https://chain.com/docs/core/learn-more/authentication-and-authorization}
 * @module AuthorizationGrantsApi
 */
const authorizationGrants = (client) => ({
  /**
   * @typedef {Object} AccessTokenGuardData
   *
   * @property {String} id
   * Unique identifier of an access token
   */

  /**
   * @typedef {Object} X509GuardData
   * x509 certificates are identified by their Subject attribute. You can
   * configure the guard by specifying values for the Subject's sub-attributes,
   * such as CN or OU. If a certificate's Subject contains all of the
   * sub-attribute values specified in the guard, the guard will produce a
   * positive match.
   *
   * @property {Object} subject - Object identifying key-value pairs in the subject field.
   * @property {(String|Array)} subject.C - Country attribute
   * @property {(String|Array)} subject.O - Organization attribute
   * @property {(String|Array)} subject.OU - Organizational Unit attribute
   * @property {(String|Array)} subject.L - Locality attribute
   * @property {(String|Array)} subject.ST - State/Province attribute
   * @property {(String|Array)} subject.STREET - Street Address attribute
   * @property {(String|Array)} subject.POSTALCODE - Postal Code attribute
   * @property {String} subject.SERIALNUMBER - Serial Number attribute
   * @property {String} subject.CN - Common Name attribute
   */

  /**
   * Create a new access grant.
   *
   * @param {Object} params - Parameters for access grant creation.
   * @param {String} params.guardType - Type of credential to guard with, either 'access_token' or 'x509'.
   * @param {Object} params.guardData - Object containing data needed to identify the incoming credential.
   * @param {String} params.policy - Authorization polciy to attach to specific grant. See {@link AuthorizationGrant} for a list of available policiies.
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Promise<Object>} Success message or error.
   */
  create: (params, cb) => {
    params = Object.assign({}, params)
    if (params.guardType == 'x509') {
      params.guardData = util.sanitizeX509GuardData(params.guardData)
    }

    return shared.create(
      client,
      '/create-authorization-grant',
      params,
      {skipArray: true, cb}
    )
  },

  /**
   * Delete the specfiied access grant.
   *
   * @param {Object} params - Parameters for access grant deletion.
   * @param {String} params.guardType - Type of credential to delete, either 'access_token' or 'x509'.
   * @param {Object} params.guardData - Object containing data needed to identify the credential to be removed.
   * @param {String} params.policy - Authorization policy to remove. See {@link AuthorizationGrant} for a list of available policiies.
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Promise<Object>} Success message or error.
   */
  delete: (params, cb) => shared.tryCallback(
    client.request('/delete-authorization-grant', params),
    cb
  ),

  /**
   * Get all access grants.
   *
   * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Promise<Array<AuthorizationGrant>>} Requested page of results.
   */
  list: (cb) =>
    shared.query(client, 'accessTokens', '/list-authorization-grants', {}, {cb}),
})

module.exports = authorizationGrants
