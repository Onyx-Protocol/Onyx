package name

// Map OID values to "short names" as registered at IANA.
// See http://www.iana.org/assignments/ldap-parameters.

var oidName = map[string]string{}

func init() {
	for name, val := range oidNumber {
		oidName[val] = name
	}
}

// This table includes only the "common" elements of a DN,
// the same set in struct pkix.Name.
var oidNumber = map[string]string{
	"C":            "2.5.4.6",
	"O":            "2.5.4.10",
	"OU":           "2.5.4.11",
	"CN":           "2.5.4.3",
	"SERIALNUMBER": "2.5.4.5",
	"L":            "2.5.4.7",
	"ST":           "2.5.4.8",
	"STREET":       "2.5.4.9",
	"POSTALCODE":   "2.5.4.17",
	"DC":           "0.9.2342.19200300.100.1.25",
	"UID":          "0.9.2342.19200300.100.1.1",
}
