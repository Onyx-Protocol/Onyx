package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;

import java.util.*;

import com.google.gson.annotations.SerializedName;

/**
 * An authorization grant provides access to resources in the Chain Core API.
 * It maps guards (predicates that match against credentials provided by
 * clients) to policies (lists of resources).
 * <p>
 * There are two types of guards: {@link AccessTokenGuard}, which matches against
 * access tokens by ID, and {@link X509Guard}, which matches against X.509 client
 * certificates that match a set of attributes.
 * <p>
 * Currently, there are four policies exposed through the API:
 * <p><ul>
 * <li>client-readwrite: full access to the Client API, including accounts,
 *   assets, transactions, access tokens, authorization grants, etc.
 * <li>client-readonly: read-only access to the Client API. API calls that modify
 *   data in Chain Core, such as account creation, are not permitted.
 * <li>monitoring: read-only access to monitoring endpoints, such as fetching
 *   the Chain Core configuration.
 * <li>crosscore: access to the cross-core API, including fetching blocks and
 *   submitting transactions, but not including block signing.
 * <li>crosscore-signblock: access to the cross-core API's block-signing API call.
 * </ul>
 */
public class AuthorizationGrant {
  @SerializedName("guard_type")
  public String guardType;

  @SerializedName("guard_data")
  public Map<String, Object> guardData;

  public String policy;

  @SerializedName("protected")
  public boolean isProtected;

  @SerializedName("created_at")
  public Date createdAt;

  /**
   * A guard that will provide access for a specific access token, identified
   * by its unique ID.
   */
  public static class AccessTokenGuard {
    public String id;

    /**
     * Specifies the ID of the token that the guard will match against.
     * @param id an access token ID (just the ID, not the full token value)
     * @return updated AccessTokenGuard object
     */
    public AccessTokenGuard setId(String id) {
      this.id = id;
      return this;
    }
  }

  /**
   * A guard that will provide access for X.509 certificates whose Subject
   * attribute matches a specified list of sub-attributes, such as CN or OU.
   * If a certificate's Subject contains all of the sub-attribute values
   * specified in the guard, the guard will produce a positive match.
   */
  public static class X509Guard {
    public static class Subject {
      @SerializedName("C")
      List<String> country;

      @SerializedName("O")
      List<String> organization;

      @SerializedName("OU")
      List<String> organizationalUnit;

      @SerializedName("L")
      List<String> locality;

      @SerializedName("ST")
      List<String> stateOrProvince;

      @SerializedName("STREET")
      List<String> streetAddress;

      @SerializedName("POSTALCODE")
      List<String> postalCode;

      @SerializedName("SERIALNUMBER")
      String serialNumber;

      @SerializedName("CN")
      String commonName;
    }

    private Subject subject;

    public X509Guard() {
      subject = new Subject();
    }

    /**
     * Specifies a value for the Country attribute. Multiple values can be specified.
     * @param s value for Country attribute
     * @return updated guard object
     */
    public X509Guard addCountry(String s) {
      if (subject.country == null) {
        subject.country = new ArrayList<>();
      }
      subject.country.add(s);
      return this;
    }

    /**
     * Specifies a value for the Organization attribute. Multiple values can be specified.
     * @param s value for Organization attribute
     * @return updated guard object
     */
    public X509Guard addOrganization(String s) {
      if (subject.organization == null) {
        subject.organization = new ArrayList<>();
      }
      subject.organization.add(s);
      return this;
    }

    /**
     * Specifies a value for the Organizational Unit attribute. Multiple values can be specified.
     * @param s value for Organizational Unit attribute
     * @return updated guard object
     */
    public X509Guard addOrganizationalUnit(String s) {
      if (subject.organizationalUnit == null) {
        subject.organizationalUnit = new ArrayList<>();
      }
      subject.organizationalUnit.add(s);
      return this;
    }

    /**
     * Specifies a value for the Locality attribute. Multiple values can be specified.
     * @param s value for Locality attribute
     * @return updated guard object
     */
    public X509Guard addLocality(String s) {
      if (subject.locality == null) {
        subject.locality = new ArrayList<>();
      }
      subject.locality.add(s);
      return this;
    }

    /**
     * Specifies a value for the State/Province attribute. Multiple values can be specified.
     * @param s value for State/Province attribute
     * @return updated guard object
     */
    public X509Guard addStateOrProvince(String s) {
      if (subject.stateOrProvince == null) {
        subject.stateOrProvince = new ArrayList<>();
      }
      subject.stateOrProvince.add(s);
      return this;
    }

    /**
     * Specifies a value for the Street Address attribute. Multiple values can be specified.
     * @param s value for Street Address attribute
     * @return updated guard object
     */
    public X509Guard addStreetAddress(String s) {
      if (subject.streetAddress == null) {
        subject.streetAddress = new ArrayList<>();
      }
      subject.streetAddress.add(s);
      return this;
    }

    /**
     * Specifies a value for the Postal Code attribute. Multiple values can be specified.
     * @param s value for Postal Code attribute
     * @return updated guard object
     */
    public X509Guard addPostalCode(String s) {
      if (subject.postalCode == null) {
        subject.postalCode = new ArrayList<>();
      }
      subject.postalCode.add(s);
      return this;
    }

    /**
     * Specifies a value for the Serial Number attribute.
     * @param s value for Serial Number attribute
     * @return updated guard object
     */
    public X509Guard setSerialNumber(String s) {
      subject.serialNumber = s;
      return this;
    }

    /**
     * Specifies a value for the Common Name attribute.
     * @param s value for Common Name attribute
     * @return updated guard object
     */
    public X509Guard setCommonName(String s) {
      subject.commonName = s;
      return this;
    }
  }

  /**
   * A base class for RPC builders that specify a grant, i.e. a
   * guard-policy tuple.
   * @param <T> always use the child class that extends BaseBuilder
   */
  public static class BaseBuilder<T> {
    @SerializedName("guard_type")
    private String guardType;

    @SerializedName("guard_data")
    private Object guardData;

    private String policy;

    /**
     * Specifies a guard that will match against a specific access token.
     * @param g a guard that matches against an access token
     * @return updated builder object
     */
    public T setGuard(AccessTokenGuard g) {
      this.guardType = "access_token";
      this.guardData = g;
      return (T) this;
    }

    /**
     * Specifies a guard that will match against X.509 certificates.
     * @param g a guard that matches against X.509 certificates
     * @return updated builder object
     */
    public T setGuard(X509Guard g) {
      guardType = "x509";
      guardData = g;
      return (T) this;
    }

    /**
     * Sets the policy to grant to credentials that match the guard.
     * @param policy One of "client-readwrite", "client-readonly", "monitoring", or "network"
     * @return updated builder object
     */
    public T setPolicy(String policy) {
      this.policy = policy;
      return (T) this;
    }
  }

  /**
   * Sets up a grant creation API call.
   */
  public static class Builder extends BaseBuilder<Builder> {
    /**
     * Creates a new grant with the parameters in this Builder instance.
     * @param client the client object providing connectivity to the Chain Core instance
     * @throws ChainException
     */
    public void create(Client client) throws ChainException {
      client.request("create-authorization-grant", this, SuccessMessage.class);
    }
  }

  /**
   * Sets up a grant deletion API call.
   */
  public static class DeletionBuilder extends BaseBuilder<DeletionBuilder> {
    /**
     * Deletes a new grant with the parameters in this Builder instance.
     * @param client the client object providing connectivity to the Chain Core instance
     * @throws ChainException
     */
    public void delete(Client client) throws ChainException {
      client.request("delete-authorization-grant", this, SuccessMessage.class);
    }
  }

  /**
   * Retrieves a list of all authorization grants in Chain Core.
   * @param client the client object providing connectivity to the Chain Core instance
   * @return a list of authorization grants
   * @throws ChainException
   */
  public static List<AuthorizationGrant> listAll(Client client) throws ChainException {
    ListResponse resp = client.request("list-authorization-grants", null, ListResponse.class);
    return resp.items;
  }

  private class ListResponse {
    public java.util.List<AuthorizationGrant> items;
  }
}
