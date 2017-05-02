package name

import (
	"crypto/x509/pkix"
	"reflect"
	"testing"
)

var (
	dc  = []int{0, 9, 2342, 19200300, 100, 1, 25}
	uid = []int{0, 9, 2342, 19200300, 100, 1, 1}
)

var cases = []struct {
	encoded string
	decoded pkix.Name
}{
	{
		`UID=jsmith,DC=example,DC=net`,
		pkix.Name{
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: dc, Value: "net"},
				{Type: dc, Value: "example"},
				{Type: uid, Value: "jsmith"},
			},
		},
	},
	{
		//`OU=Sales+CN=J. Smith,DC=example,DC=net`,
		`DC=example,DC=net,CN=J. Smith,OU=Sales`,
		pkix.Name{
			CommonName:         "J. Smith",
			OrganizationalUnit: []string{"Sales"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: dc, Value: "net"},
				{Type: dc, Value: "example"},
			},
		},
	},
	{
		//`CN=James \"Jim\" Smith\, III,DC=example,DC=net`,
		`DC=example,DC=net,CN=James \"Jim\" Smith\, III`,
		pkix.Name{
			CommonName: `James "Jim" Smith, III`,
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: dc, Value: "net"},
				{Type: dc, Value: "example"},
			},
		},
	},
	{
		//"CN=Before\nAfter,DC=example,DC=net",
		"DC=example,DC=net,CN=Before\nAfter",
		pkix.Name{
			CommonName: "Before\nAfter",
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: dc, Value: "net"},
				{Type: dc, Value: "example"},
			},
		},
	},
	{
		`1.3.6.1.4.1.1466.0=#04024869`,
		pkix.Name{
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: []int{1, 3, 6, 1, 4, 1, 1466, 0}, Value: []byte{0x48, 0x69}},
			},
		},
	},
	{
		`1.3.6.1.4.1.1466.0=#04024869,O=Test,C=GB`,
		pkix.Name{
			Country:      []string{"GB"},
			Organization: []string{"Test"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{Type: []int{1, 3, 6, 1, 4, 1, 1466, 0}, Value: []byte{0x48, 0x69}},
			},
		},
	},
	{
		`CN=Lučić`,
		pkix.Name{CommonName: "Lučić"},
	},
	{
		`OU=West+OU=Engineering,O=Acme Corp.`,
		pkix.Name{
			Organization:       []string{"Acme Corp."},
			OrganizationalUnit: []string{"West", "Engineering"},
		},
	},
	{
		`CN=localhost,O=Chain\, Inc.`,
		pkix.Name{
			Organization: []string{"Chain, Inc."},
			CommonName:   "localhost",
		},
	},
}

func TestFormat(t *testing.T) {
	for _, test := range cases {
		got, err := Format(test.decoded)
		if err != nil {
			t.Errorf("Format(%+v) err = %v, want nil", test.decoded, err)
			continue
		}
		if got != test.encoded {
			t.Errorf("Format(%+v) = %#q, want %#q", test.decoded, got, test.encoded)
		}
	}
}

func TestParse(t *testing.T) {
	for _, test := range cases {
		t.Run(test.encoded, func(t *testing.T) {
			t.Logf("Parse(%q)", test.encoded)
			got, err := Parse(test.encoded)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			got.Names = nil
			test.decoded.ExtraNames = nil // pkix ignore unknown entries; doesn't put them here
			if !reflect.DeepEqual(got, test.decoded) {
				t.Errorf("got %+v", got)
				t.Logf("want %+v", test.decoded)
			}
		})
	}
}
