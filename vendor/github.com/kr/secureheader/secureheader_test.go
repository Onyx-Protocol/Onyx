package secureheader

import (
	"testing"
)

func TestDefaultUseForwardedProto(t *testing.T) {
	if defaultUseForwardedProto {
		t.Fatal("defaultUseForwardedProto = true want false")
	}
}
