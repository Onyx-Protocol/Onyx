package authz

import (
	"crypto/x509/pkix"
	"encoding/json"
	"strings"
)

// Same as crypto/x509/pkix.Name but with JSON tags.
type pkixDN struct {
	Country            []string `json:"C,omitempty"`
	Organization       []string `json:"O,omitempty"`
	OrganizationalUnit []string `json:"OU,omitempty"`
	Locality           []string `json:"L,omitempty"`
	Province           []string `json:"ST,omitempty"`
	StreetAddress      []string `json:"STREET,omitempty"`
	PostalCode         []string `json:"POSTALCODE,omitempty"`
	SerialNumber       string   `json:"SERIALNUMBER,omitempty"`
	CommonName         string   `json:"CN,omitempty"`

	Names      []pkix.AttributeTypeAndValue `json:"-"`
	ExtraNames []pkix.AttributeTypeAndValue `json:"-"`
}

var x509FieldNames = map[string]bool{
	"C":            true,
	"O":            true,
	"OU":           true,
	"L":            true,
	"ST":           true,
	"STREET":       true,
	"POSTALCODE":   true,
	"SERIALNUMBER": true,
	"CN":           true,
}

func ValidX509SubjectField(s string) bool {
	return x509FieldNames[strings.ToUpper(s)]
}

func x509GuardData(data []byte) pkix.Name {
	var v struct{ Subject pkixDN }
	err := json.Unmarshal(data, &v)
	if err != nil {
		// We should create only well-formed guard data,
		// so this should not happen.
		// (And if it does, it's our bug.)
		panic(err)
	}
	return pkix.Name(v.Subject)
}

func encodeX509GuardData(subj pkix.Name) []byte {
	v := struct {
		Subject pkixDN `json:"subject"`
	}{pkixDN(subj)}
	d, _ := json.Marshal(v)
	return d
}

func matchesX509(pat, x pkix.Name) bool {
	return matchesString(pat.CommonName, x.CommonName) &&
		matchesStrings(pat.Country, x.Country) &&
		matchesStrings(pat.Organization, x.Organization) &&
		matchesStrings(pat.OrganizationalUnit, x.OrganizationalUnit) &&
		matchesStrings(pat.Locality, x.Locality) &&
		matchesStrings(pat.Province, x.Province) &&
		matchesStrings(pat.StreetAddress, x.StreetAddress) &&
		matchesStrings(pat.PostalCode, x.PostalCode) &&
		matchesString(pat.SerialNumber, x.SerialNumber)
}

func matchesStrings(pat, x []string) bool {
	if len(pat) == 0 {
		return true
	} else if len(x) != len(pat) {
		return false
	}
	for i, s := range x {
		if s != pat[i] {
			return false
		}
	}
	return true
}

func matchesString(pat, x string) bool {
	return pat == "" || pat == x
}
