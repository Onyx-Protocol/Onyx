//+build loopback_auth

package authz

func init() {
	builtinGrants = append(builtinGrants,
		&Grant{GuardType: "localhost", Policy: "client-readwrite"},
		&Grant{GuardType: "localhost", Policy: "client-readonly"},
		&Grant{GuardType: "localhost", Policy: "monitoring"},
		&Grant{GuardType: "localhost", Policy: "crosscore"},
		&Grant{GuardType: "localhost", Policy: "internal"},
	)
}
