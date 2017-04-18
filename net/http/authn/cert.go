package authn

type CertGuardData struct {
	Subject struct {
		CommonName         string   `json:"cn"`
		OrganizationalUnit []string `json:"ou"`
	}
}

func (c *CertGuardData) Equals(other *CertGuardData) bool {
	if other == nil {
		return false
	}
	if c.Subject.CommonName != other.Subject.CommonName {
		return false
	}
	if len(c.Subject.OrganizationalUnit) != len(other.Subject.OrganizationalUnit) {
		return false
	}
	for i := range c.Subject.OrganizationalUnit {
		if c.Subject.OrganizationalUnit[i] != other.Subject.OrganizationalUnit[i] {
			return false
		}
	}
	return true
}
