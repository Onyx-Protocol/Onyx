package chainkd

import (
	"log"
	"testing"
)

var (
	benchXprv XPrv
	benchXpub XPub
	benchMsg  = []byte("Hello, world!")
	benchSig  []byte
)

func init() {
	var err error
	benchXprv, err = NewXPrv(nil)
	if err != nil {
		log.Fatalln(err)
	}
	benchXpub = benchXprv.XPub()
	benchSig = benchXprv.Sign(benchMsg)
}

func BenchmarkXPrvChildNonHardened(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchXprv.Child(benchMsg, false)
	}
}

func BenchmarkXPrvChildHardened(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchXprv.Child(benchMsg, true)
	}
}

func BenchmarkXPubChild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchXpub.Child(benchMsg)
	}
}

func BenchmarkXPrvSign(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchXprv.Sign(benchMsg)
	}
}

func BenchmarkXPubVerify(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchXpub.Verify(benchMsg, benchSig)
	}
}
