package authn

type CertGuardData struct {
	Subject struct {
		CommonName         string   `json:"cn"`
		OrganizationalUnit []string `json:"ou"`
	}
}

func (sn *CertGuardData) Equals(other *CertGuardData) bool {
	if other == nil {
		return false
	}
	if sn.Subject.CommonName != other.Subject.CommonName {
		return false
	}
	if len(sn.Subject.OrganizationalUnit) != len(other.Subject.OrganizationalUnit) {
		return false
	}
	for i := range sn.Subject.OrganizationalUnit {
		if sn.Subject.OrganizationalUnit[i] != other.Subject.OrganizationalUnit[i] {
			return false
		}
	}
	return true
}
