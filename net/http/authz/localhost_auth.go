//+build localhost_auth

package authz

func init() {
	builtinGrants = append(builtinGrants,
		// TODO(kr): refactor this list/logic somehow to get data
		// from the source of truth for all policies.
		&Grant{GuardType: "localhost", Policy: "client-readwrite"},
		&Grant{GuardType: "localhost", Policy: "client-readonly"},
		&Grant{GuardType: "localhost", Policy: "monitoring"},
		&Grant{GuardType: "localhost", Policy: "crosscore"},
		&Grant{GuardType: "localhost", Policy: "crosscore-signblock"},
		&Grant{GuardType: "localhost", Policy: "internal"},
	)
}
