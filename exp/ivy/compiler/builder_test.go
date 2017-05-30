package compiler

// func TestBuilder(t *testing.T) {
// 	cases := []struct {
// 		name    string
// 		f       func(*builder)
// 		wantHex string
// 	}{
// 		{
// 			"single pushdata",
// 			func(b *builder) {
// 				b.addInt64(1)
// 			},
// 			"51",
// 		},
// 		{
// 			"pushdata and verify",
// 			func(b *builder) {
// 				b.addInt64(1)
// 				b.addOp(vm.OP_VERIFY)
// 			},
// 			"51",
// 		},
// 		{
// 			"pushdata, verify, second pushdata",
// 			func(b *builder) {
// 				b.addInt64(1)
// 				b.addOp(vm.OP_VERIFY)
// 				b.addInt64(2)
// 			},
// 			"516952",
// 		},
// 	}
// 	for _, c := range cases {
// 		t.Run(c.name, func(t *testing.T) {
// 			b := newBuilder()
// 			c.f(b)
// 			got, err := b.build()
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			want, err := hex.DecodeString(c.wantHex)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			if !bytes.Equal(got, want) {
// 				t.Errorf("got %x, want %x", got, want)
// 			}
// 		})
// 	}
// }
