//+build !loopback_auth

package authz

func init() {
	builtinGrants = append(builtinGrants,
		&Grant{GuardType: "any", Policy: "public"},
	)
}
