const shared = require('../shared')

/**
 * Authorization grants provide a mapping from guard objects (access tokens or X509
 * certificates) to a list of predefined Chain Core access policies.
 *
 * * **client-readwrite**: full access to the Client API
 * * **client-readonly**: access to read-only Client endpoints
 * * **network**: access to the Network API
 * * **monitoring**: access to monitoring-specific endpoints
 *
 * More info: {@link https://chain.com/docs/core/learn-more/authentication}
 * @typedef {Object} AuthorizationGrant
 * @global
 *
 * @property {String} guard_type
 * Type of credential, either 'access_token' or 'x509'.
 *
 * @property {Object} guard_data
 * Data used by the guard to identity incoming credentials.
 *
 * If guard_type is 'access_token', you should provide an instance of
 * {@link module:AuthorizationGrantsApi~AccessTokenGuardData}, which identifies access tokens by ID.
 *
 * If guard_type is 'x509', you should provide an instance of {@link module:AuthorizationGrantsApi~X509GuardData},
 * which identifies x509 certificates based on kev-value pairs in specified
 * certificate fields.
 *
 * @property {String} policy
 * Authorization single polciy to attach to specific grant.
 *
 * @property {String} createdAt
 * Time of grant creation, RFC3339 formatted.
 */

/**
 * API for interacting with {@link AuthorizationGrant access grants}.
 *
 * More info: {@link https://chain.com/docs/core/learn-more/authentication}
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
   * x509 certificates can be identified by any field. As an example, the
   * properties on this type document the `subject` field below:
   *
   * ```
   * CN=Alice, OU=Engineering
   * ```
   *
   * Fields other than `subject`, and keys other than `cn` and `ou`, can
   * be addressed in a similar manner.
   *
   * @property {Object} subject - Object identifying key-value pairs in the subject field.
   * @property {String} subject.cn - "Common Name" to match against.
   * @property {String} subject.ou - "Organizational Unit" to match against.
   */

  /**
   * Create a new access grant.
   *
   * @param {Object} params - Parameters for access grant creation.
   * @param {String} params.guard_type - Type of credential to guard with, either 'access_token' or 'x509'.
   * @param {Object} params.guard_data - Object containing data needed to identify the incoming credential.
   * @param {String} params.policy - Authorization polciy to attach to specific grant. See {@link AuthorizationGrant} for a list of available policiies.
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Promise<Object>} Success message or error.
   */
  create: (params , cb) =>
    shared.create(client, '/create-authorization-grant', params, {skipArray: true, cb}),

  /**
   * Delete the specfiied access grant.
   *
   * @param {Object} params - Parameters for access grant deletion.
   * @param {String} params.guard_type - Type of credential to delete, either 'access_token' or 'x509'.
   * @param {Object} params.guard_data - Object containing data needed to identify the credential to be removed.
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
