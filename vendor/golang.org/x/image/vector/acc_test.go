// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"bytes"
	"fmt"
	"testing"
)

// TestXxxSIMDUnaligned tests that unaligned SIMD loads/stores don't crash.

func TestFixedAccumulateSIMDUnaligned(t *testing.T) {
	if !haveFixedAccumulateSIMD {
		t.Skip("No SIMD implemention")
	}

	dst := make([]uint8, 64)
	src := make([]uint32, 64)
	for d := 0; d < 16; d++ {
		for s := 0; s < 16; s++ {
			fixedAccumulateOpSrcSIMD(dst[d:d+32], src[s:s+32])
		}
	}
}

func TestFloatingAccumulateSIMDUnaligned(t *testing.T) {
	if !haveFloatingAccumulateSIMD {
		t.Skip("No SIMD implemention")
	}

	dst := make([]uint8, 64)
	src := make([]float32, 64)
	for d := 0; d < 16; d++ {
		for s := 0; s < 16; s++ {
			floatingAccumulateOpSrcSIMD(dst[d:d+32], src[s:s+32])
		}
	}
}

// TestXxxSIMDShortDst tests that the SIMD implementations don't write past the
// end of the dst buffer.

func TestFixedAccumulateSIMDShortDst(t *testing.T) {
	if !haveFixedAccumulateSIMD {
		t.Skip("No SIMD implemention")
	}

	const oneQuarter = uint32(int2ϕ(fxOne*fxOne)) / 4
	src := []uint32{oneQuarter, oneQuarter, oneQuarter, oneQuarter}
	for i := 0; i < 4; i++ {
		dst := make([]uint8, 4)
		fixedAccumulateOpSrcSIMD(dst[:i], src[:i])
		for j := range dst {
			if j < i {
				if got := dst[j]; got == 0 {
					t.Errorf("i=%d, j=%d: got %#02x, want non-zero", i, j, got)
				}
			} else {
				if got := dst[j]; got != 0 {
					t.Errorf("i=%d, j=%d: got %#02x, want zero", i, j, got)
				}
			}
		}
	}
}

func TestFloatingAccumulateSIMDShortDst(t *testing.T) {
	if !haveFloatingAccumulateSIMD {
		t.Skip("No SIMD implemention")
	}

	const oneQuarter = 0.25
	src := []float32{oneQuarter, oneQuarter, oneQuarter, oneQuarter}
	for i := 0; i < 4; i++ {
		dst := make([]uint8, 4)
		floatingAccumulateOpSrcSIMD(dst[:i], src[:i])
		for j := range dst {
			if j < i {
				if got := dst[j]; got == 0 {
					t.Errorf("i=%d, j=%d: got %#02x, want non-zero", i, j, got)
				}
			} else {
				if got := dst[j]; got != 0 {
					t.Errorf("i=%d, j=%d: got %#02x, want zero", i, j, got)
				}
			}
		}
	}
}

func TestFixedAccumulateOpOverShort(t *testing.T)    { testAcc(t, fxInShort, fxMaskShort, "over") }
func TestFixedAccumulateOpSrcShort(t *testing.T)     { testAcc(t, fxInShort, fxMaskShort, "src") }
func TestFixedAccumulateMaskShort(t *testing.T)      { testAcc(t, fxInShort, fxMaskShort, "mask") }
func TestFloatingAccumulateOpOverShort(t *testing.T) { testAcc(t, flInShort, flMaskShort, "over") }
func TestFloatingAccumulateOpSrcShort(t *testing.T)  { testAcc(t, flInShort, flMaskShort, "src") }
func TestFloatingAccumulateMaskShort(t *testing.T)   { testAcc(t, flInShort, flMaskShort, "mask") }

func TestFixedAccumulateOpOver16(t *testing.T)    { testAcc(t, fxIn16, fxMask16, "over") }
func TestFixedAccumulateOpSrc16(t *testing.T)     { testAcc(t, fxIn16, fxMask16, "src") }
func TestFixedAccumulateMask16(t *testing.T)      { testAcc(t, fxIn16, fxMask16, "mask") }
func TestFloatingAccumulateOpOver16(t *testing.T) { testAcc(t, flIn16, flMask16, "over") }
func TestFloatingAccumulateOpSrc16(t *testing.T)  { testAcc(t, flIn16, flMask16, "src") }
func TestFloatingAccumulateMask16(t *testing.T)   { testAcc(t, flIn16, flMask16, "mask") }

func testAcc(t *testing.T, in interface{}, mask []uint32, op string) {
	for _, simd := range []bool{false, true} {
		maxN := 0
		switch in := in.(type) {
		case []uint32:
			if simd && !haveFixedAccumulateSIMD {
				continue
			}
			maxN = len(in)
		case []float32:
			if simd && !haveFloatingAccumulateSIMD {
				continue
			}
			maxN = len(in)
		}

		for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
			33, 55, 79, 96, 120, 165, 256, maxN} {

			if n > maxN {
				continue
			}

			var (
				got8, want8   []uint8
				got32, want32 []uint32
			)
			switch op {
			case "over":
				const background = 0x40
				got8 = make([]uint8, n)
				for i := range got8 {
					got8[i] = background
				}
				want8 = make([]uint8, n)
				for i := range want8 {
					dstA := uint32(background * 0x101)
					maskA := mask[i]
					outA := dstA*(0xffff-maskA)/0xffff + maskA
					want8[i] = uint8(outA >> 8)
				}

			case "src":
				got8 = make([]uint8, n)
				want8 = make([]uint8, n)
				for i := range want8 {
					want8[i] = uint8(mask[i] >> 8)
				}

			case "mask":
				got32 = make([]uint32, n)
				want32 = mask[:n]
			}

			switch in := in.(type) {
			case []uint32:
				switch op {
				case "over":
					fixedAccumulateOpOver(got8, in[:n])
				case "src":
					if simd {
						fixedAccumulateOpSrcSIMD(got8, in[:n])
					} else {
						fixedAccumulateOpSrc(got8, in[:n])
					}
				case "mask":
					copy(got32, in[:n])
					fixedAccumulateMask(got32)
				}
			case []float32:
				switch op {
				case "over":
					floatingAccumulateOpOver(got8, in[:n])
				case "src":
					if simd {
						floatingAccumulateOpSrcSIMD(got8, in[:n])
					} else {
						floatingAccumulateOpSrc(got8, in[:n])
					}
				case "mask":
					floatingAccumulateMask(got32, in[:n])
				}
			}

			if op != "mask" {
				if !bytes.Equal(got8, want8) {
					t.Errorf("simd=%t, n=%d:\ngot:  % x\nwant: % x", simd, n, got8, want8)
				}
			} else {
				if !uint32sEqual(got32, want32) {
					t.Errorf("simd=%t, n=%d:\ngot:  % x\nwant: % x", simd, n, got32, want32)
				}
			}
		}
	}
}

func uint32sEqual(xs, ys []uint32) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i := range xs {
		if xs[i] != ys[i] {
			return false
		}
	}
	return true
}

func float32sEqual(xs, ys []float32) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i := range xs {
		if xs[i] != ys[i] {
			return false
		}
	}
	return true
}

func BenchmarkFixedAccumulateOpOver16(b *testing.B)       { benchAcc(b, fxIn16, "over", false) }
func BenchmarkFixedAccumulateOpSrc16(b *testing.B)        { benchAcc(b, fxIn16, "src", false) }
func BenchmarkFixedAccumulateOpSrcSIMD16(b *testing.B)    { benchAcc(b, fxIn16, "src", true) }
func BenchmarkFixedAccumulateMask16(b *testing.B)         { benchAcc(b, fxIn16, "mask", false) }
func BenchmarkFloatingAccumulateOpOver16(b *testing.B)    { benchAcc(b, flIn16, "over", false) }
func BenchmarkFloatingAccumulateOpSrc16(b *testing.B)     { benchAcc(b, flIn16, "src", false) }
func BenchmarkFloatingAccumulateOpSrcSIMD16(b *testing.B) { benchAcc(b, flIn16, "src", true) }
func BenchmarkFloatingAccumulateMask16(b *testing.B)      { benchAcc(b, flIn16, "mask", false) }

func BenchmarkFixedAccumulateOpOver64(b *testing.B)       { benchAcc(b, fxIn64, "over", false) }
func BenchmarkFixedAccumulateOpSrc64(b *testing.B)        { benchAcc(b, fxIn64, "src", false) }
func BenchmarkFixedAccumulateOpSrcSIMD64(b *testing.B)    { benchAcc(b, fxIn64, "src", true) }
func BenchmarkFixedAccumulateMask64(b *testing.B)         { benchAcc(b, fxIn64, "mask", false) }
func BenchmarkFloatingAccumulateOpOver64(b *testing.B)    { benchAcc(b, flIn64, "over", false) }
func BenchmarkFloatingAccumulateOpSrc64(b *testing.B)     { benchAcc(b, flIn64, "src", false) }
func BenchmarkFloatingAccumulateOpSrcSIMD64(b *testing.B) { benchAcc(b, flIn64, "src", true) }
func BenchmarkFloatingAccumulateMask64(b *testing.B)      { benchAcc(b, flIn64, "mask", false) }

func benchAcc(b *testing.B, in interface{}, op string, simd bool) {
	var f func()

	switch in := in.(type) {
	case []uint32:
		if simd && !haveFixedAccumulateSIMD {
			b.Skip("No SIMD implemention")
		}

		switch op {
		case "over":
			dst := make([]uint8, len(in))
			f = func() { fixedAccumulateOpOver(dst, in) }
		case "src":
			dst := make([]uint8, len(in))
			if simd {
				f = func() { fixedAccumulateOpSrcSIMD(dst, in) }
			} else {
				f = func() { fixedAccumulateOpSrc(dst, in) }
			}
		case "mask":
			buf := make([]uint32, len(in))
			copy(buf, in)
			f = func() { fixedAccumulateMask(buf) }
		}

	case []float32:
		if simd && !haveFloatingAccumulateSIMD {
			b.Skip("No SIMD implemention")
		}

		switch op {
		case "over":
			dst := make([]uint8, len(in))
			f = func() { floatingAccumulateOpOver(dst, in) }
		case "src":
			dst := make([]uint8, len(in))
			if simd {
				f = func() { floatingAccumulateOpSrcSIMD(dst, in) }
			} else {
				f = func() { floatingAccumulateOpSrc(dst, in) }
			}
		case "mask":
			dst := make([]uint32, len(in))
			f = func() { floatingAccumulateMask(dst, in) }
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

// itou exists because "uint32(int2ϕ(-1))" doesn't compile: constant -1
// overflows uint32.
func itou(i int2ϕ) uint32 {
	return uint32(i)
}

var fxInShort = []uint32{
	itou(+0x020000), // +0.125, // Running sum: +0.125
	itou(-0x080000), // -0.500, // Running sum: -0.375
	itou(+0x040000), // +0.250, // Running sum: -0.125
	itou(+0x060000), // +0.375, // Running sum: +0.250
	itou(+0x020000), // +0.125, // Running sum: +0.375
	itou(+0x000000), // +0.000, // Running sum: +0.375
	itou(-0x100000), // -1.000, // Running sum: -0.625
	itou(-0x080000), // -0.500, // Running sum: -1.125
	itou(+0x040000), // +0.250, // Running sum: -0.875
	itou(+0x0e0000), // +0.875, // Running sum: +0.000
	itou(+0x040000), // +0.250, // Running sum: +0.250
	itou(+0x0c0000), // +0.750, // Running sum: +1.000
}

var flInShort = []float32{
	+0.125, // Running sum: +0.125
	-0.500, // Running sum: -0.375
	+0.250, // Running sum: -0.125
	+0.375, // Running sum: +0.250
	+0.125, // Running sum: +0.375
	+0.000, // Running sum: +0.375
	-1.000, // Running sum: -0.625
	-0.500, // Running sum: -1.125
	+0.250, // Running sum: -0.875
	+0.875, // Running sum: +0.000
	+0.250, // Running sum: +0.250
	+0.750, // Running sum: +1.000
}

// It's OK for fxMaskShort and flMaskShort to have slightly different values.
// Both the fixed and floating point implementations already have (different)
// rounding errors in the xxxLineTo methods before we get to accumulation. It's
// OK for 50% coverage (in ideal math) to be approximated by either 0x7fff or
// 0x8000. Both slices do contain checks that 0% and 100% map to 0x0000 and
// 0xffff, as does checkCornersCenter in vector_test.go.
//
// It is important, though, for the SIMD and non-SIMD fixed point
// implementations to give the exact same output, and likewise for the floating
// point implementations.

var fxMaskShort = []uint32{
	0x2000,
	0x6000,
	0x2000,
	0x4000,
	0x6000,
	0x6000,
	0xa000,
	0xffff,
	0xe000,
	0x0000,
	0x4000,
	0xffff,
}

var flMaskShort = []uint32{
	0x1fff,
	0x5fff,
	0x1fff,
	0x3fff,
	0x5fff,
	0x5fff,
	0x9fff,
	0xffff,
	0xdfff,
	0x0000,
	0x3fff,
	0xffff,
}

func TestMakeFxInXxx(t *testing.T) {
	dump := func(us []uint32) string {
		var b bytes.Buffer
		for i, u := range us {
			if i%8 == 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%#08x, ", u)
		}
		return b.String()
	}

	if !uint32sEqual(fxIn16, hardCodedFxIn16) {
		t.Errorf("height 16: got:%v\nwant:%v", dump(fxIn16), dump(hardCodedFxIn16))
	}
}

func TestMakeFlInXxx(t *testing.T) {
	dump := func(fs []float32) string {
		var b bytes.Buffer
		for i, f := range fs {
			if i%8 == 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%v, ", f)
		}
		return b.String()
	}

	if !float32sEqual(flIn16, hardCodedFlIn16) {
		t.Errorf("height 16: got:%v\nwant:%v", dump(flIn16), dump(hardCodedFlIn16))
	}
}

func makeInXxx(height int, useFloatingPointMath bool) *Rasterizer {
	width, data := scaledBenchmarkGlyphData(height)
	z := NewRasterizer(width, height)
	z.setUseFloatingPointMath(useFloatingPointMath)
	for _, d := range data {
		switch d.n {
		case 0:
			z.MoveTo(d.p)
		case 1:
			z.LineTo(d.p)
		case 2:
			z.QuadTo(d.p, d.q)
		}
	}
	return z
}

func makeFxInXxx(height int) []uint32 {
	z := makeInXxx(height, false)
	return z.bufU32
}

func makeFlInXxx(height int) []float32 {
	z := makeInXxx(height, true)
	return z.bufF32
}

// fxInXxx and flInXxx are the z.bufU32 and z.bufF32 inputs to the accumulate
// functions when rasterizing benchmarkGlyphData at a height of Xxx pixels.
//
// fxMaskXxx and flMaskXxx are the corresponding golden outputs of those
// accumulateMask functions.
//
// The hardCodedEtc versions are a sanity check for unexpected changes in the
// rasterization implementations up to but not including accumulation.

var (
	fxIn16 = makeFxInXxx(16)
	fxIn64 = makeFxInXxx(64)
	flIn16 = makeFlInXxx(16)
	flIn64 = makeFlInXxx(64)
)

var hardCodedFxIn16 = []uint32{
	0x00000000, 0x00000000, 0xffffa3ee, 0xfff9f0c9, 0xfffaaafc, 0xfffd38ec, 0xffff073f, 0x0001dddf,
	0x0002589a, 0x0006a22c, 0x0004a6df, 0x000000a0, 0x00000000, 0x00000000, 0xfffdb883, 0xfff4c620,
	0xfffd815f, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00052ec6,
	0x000ab1ba, 0x00001f7f, 0xffff29b7, 0xfff2ad44, 0xfffe2906, 0x00006c84, 0x0006ce82, 0x00050d7b,
	0x00010db4, 0xfffd8c05, 0xfff85159, 0xfffccc6d, 0x00000000, 0x00088d28, 0x000772d8, 0xfff8a36a,
	0xfff75c96, 0x00000000, 0x000a2b80, 0x0005d480, 0x00000000, 0x00000000, 0x00000000, 0xffff4bbf,
	0xfff2b937, 0xfffdfb0b, 0x0001cc00, 0x000e3400, 0xfffa4980, 0xfffcb680, 0x000008e8, 0x0008966f,
	0x000060a8, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xfff72000, 0xfff8e000, 0x00000165,
	0x000e9134, 0x00016d65, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0xfff8d3be, 0xfff72c42, 0x00000000, 0x000cec0f, 0x000313f1, 0x00000000,
	0x00000000, 0xfffe84f1, 0xfffbbb8f, 0xfffe3008, 0xfffe311b, 0xffff1e60, 0x00000000, 0xfffd6f10,
	0xfffcd0f0, 0x00000000, 0x000cec00, 0x00031400, 0xfffe6d8a, 0xfff7d307, 0xfffa38bf, 0xffff86b3,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x000cec00,
	0x00024dc4, 0xfff3cc79, 0xfffcf9c4, 0x00003ed0, 0x000467df, 0x0004c32f, 0x0001a038, 0x00012964,
	0x00002883, 0xfffa7bf1, 0xfff9280f, 0x00000000, 0x000cec00, 0xfffa2901, 0xfff8eaff, 0x00004138,
	0x000aebd5, 0x0004d2f2, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xfff8dc00, 0xfff72400,
	0x00000000, 0x000cec00, 0xfff64800, 0xfffccc00, 0x00039400, 0x000c6c00, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0xfff8dc00, 0xfff72400, 0x00000000, 0x000cec00, 0xfff3ea8a,
	0xffff2976, 0x00047cad, 0x000b8353, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0xfff6cb2e, 0xfff934d2, 0x00000000, 0x000cec00, 0xfff68000, 0xfffc9400, 0x0000babf, 0x000cfbcc,
	0x00024974, 0x00000000, 0x00000000, 0x00000000, 0xfffa12a1, 0xfff61e13, 0xffffcf4d, 0x00000000,
	0x000c79a0, 0xfffcac8c, 0xfff6d9d4, 0x00000000, 0x00015024, 0x0006d297, 0x000288dc, 0xfffe8e52,
	0xfffaba3a, 0xfffc0cbd, 0xffffff20, 0x00000000, 0x00000000, 0x000b5c00, 0x000496d7, 0xfff5a25f,
	0xfffa6acc, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x0002abf1, 0x0005195f,
	0xfff83aae, 0x00000000, 0x00089fb6, 0x0007604a, 0xfffffe47, 0xfffb0173, 0xfff94d6b, 0xfffd7586,
	0xffff5219, 0x000319cc, 0x0003eed3, 0x0007529f, 0xfffedc08, 0xfff647f6, 0x00000000, 0x000392ce,
}

var hardCodedFlIn16 = []float32{
	0, 0, -0.022306755, -0.3782405, -0.33334962, -0.1741521, -0.0607556, 0.11660573,
	0.14664596, 0.41462868, 0.2907673, 0.0001568835, 0, 0, -0.14239307, -0.7012868,
	-0.15632017, 0, 0, 0, 0, 0, 0, 0.3230303,
	0.6690931, 0.007876594, -0.05189419, -0.832786, -0.11531975, 0.026225802, 0.42518616, 0.3154636,
	0.06598757, -0.15304244, -0.47969276, -0.20012794, 0, 0.5327272, 0.46727282, -0.45950258,
	-0.5404974, 0, 0.63484025, 0.36515975, 0, 0, 0, -0.04351709,
	-0.8293345, -0.12714837, 0.11087036, 0.88912964, -0.35792422, -0.2053554, 0.0022513224, 0.5374398,
	0.023588525, 0, 0, 0, 0, -0.55346966, -0.44653034, 0.0002531938,
	0.9088273, 0.090919495, 0, 0, 0, 0, 0, 0,
	0, 0, -0.44745448, -0.5525455, 0, 0.80748945, 0.19251058, 0,
	0, -0.092476256, -0.2661464, -0.11322958, -0.11298219, -0.055094406, 0, -0.16045958,
	-0.1996116, 0, 0.80748653, 0.19251347, -0.09804727, -0.51129663, -0.3610403, -0.029615778,
	0, 0, 0, 0, 0, 0, 0, 0.80748653,
	0.14411622, -0.76251525, -0.1890875, 0.01527351, 0.27528667, 0.29730347, 0.101477206, 0.07259522,
	0.009900213, -0.34395567, -0.42788061, 0, 0.80748653, -0.3648737, -0.44261283, 0.015778137,
	0.6826565, 0.30156538, 0, 0, 0, 0, -0.44563293, -0.55436707,
	0, 0.80748653, -0.60703933, -0.20044717, 0.22371745, 0.77628255, 0, 0,
	0, 0, 0, -0.44563293, -0.55436707, 0, 0.80748653, -0.7550391,
	-0.05244744, 0.2797074, 0.72029257, 0, 0, 0, 0, 0,
	-0.57440215, -0.42559785, 0, 0.80748653, -0.59273535, -0.21475118, 0.04544862, 0.81148535,
	0.14306602, 0, 0, 0, -0.369642, -0.61841226, -0.011945802, 0,
	0.7791623, -0.20691396, -0.57224834, 0, 0.08218567, 0.42637306, 0.1586175, -0.089709565,
	-0.32935485, -0.24788953, -0.00022224105, 0, 0, 0.7085409, 0.28821066, -0.64765793,
	-0.34909368, 0, 0, 0, 0, 0, 0.16679136, 0.31914657,
	-0.48593786, 0, 0.537915, 0.462085, -0.00041967133, -0.3120329, -0.41914812, -0.15886839,
	-0.042683028, 0.19370951, 0.24624406, 0.45803425, -0.07049577, -0.6091341, 0, 0.22253075,
}

var fxMask16 = []uint32{
	0x0000, 0x0000, 0x05c1, 0x66b4, 0xbc04, 0xe876, 0xf802, 0xda24, 0xb49a, 0x4a77, 0x0009, 0x0000, 0x0000,
	0x0000, 0x2477, 0xd815, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xad13, 0x01f7, 0x0000,
	0x0d64, 0xe290, 0xffff, 0xf937, 0x8c4f, 0x3b77, 0x2a9c, 0x51dc, 0xccc6, 0xffff, 0xffff, 0x772d, 0x0000,
	0x75c9, 0xffff, 0xffff, 0x5d47, 0x0000, 0x0000, 0x0000, 0x0000, 0x0b43, 0xdfb0, 0xffff, 0xe33f, 0x0000,
	0x5b67, 0x8fff, 0x8f71, 0x060a, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x8dff, 0xffff, 0xffe9, 0x16d6,
	0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x72c4, 0xffff, 0xffff, 0x313e,
	0x0000, 0x0000, 0x0000, 0x17b0, 0x5bf7, 0x78f7, 0x95e5, 0xa3ff, 0xa3ff, 0xcd0e, 0xffff, 0xffff, 0x313f,
	0x0000, 0x1927, 0x9bf6, 0xf86a, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0x313f,
	0x0c63, 0xcf9b, 0xffff, 0xfc12, 0xb594, 0x6961, 0x4f5e, 0x3cc7, 0x3a3f, 0x9280, 0xffff, 0xffff, 0x313f,
	0x8eaf, 0xffff, 0xfbec, 0x4d2e, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x723f, 0xffff, 0xffff, 0x313f,
	0xccbf, 0xffff, 0xc6bf, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x723f, 0xffff, 0xffff, 0x313f,
	0xf297, 0xffff, 0xb834, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x934c, 0xffff, 0xffff, 0x313f,
	0xc93f, 0xffff, 0xf453, 0x2497, 0x0000, 0x0000, 0x0000, 0x0000, 0x5ed5, 0xfcf4, 0xffff, 0xffff, 0x3865,
	0x6d9c, 0xffff, 0xffff, 0xeafd, 0x7dd4, 0x5546, 0x6c61, 0xc0bd, 0xfff1, 0xffff, 0xffff, 0xffff, 0x4a3f,
	0x00d2, 0xa6ac, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xd540, 0x83aa, 0xffff, 0xffff, 0x7604,
	0x0000, 0x001b, 0x5004, 0xbb2d, 0xe3d5, 0xeeb3, 0xbd16, 0x7e29, 0x08ff, 0x1b3f, 0xb6bf, 0xb6bf, 0x7d92,
}

var flMask16 = []uint32{
	0x0000, 0x0000, 0x05b5, 0x668a, 0xbbe0, 0xe875, 0xf803, 0xda29, 0xb49f, 0x4a7a, 0x000a, 0x0000, 0x0000,
	0x0000, 0x2473, 0xd7fb, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xad4d, 0x0204, 0x0000,
	0x0d48, 0xe27a, 0xffff, 0xf949, 0x8c70, 0x3bae, 0x2ac9, 0x51f7, 0xccc4, 0xffff, 0xffff, 0x779f, 0x0000,
	0x75a1, 0xffff, 0xffff, 0x5d7b, 0x0000, 0x0000, 0x0000, 0x0000, 0x0b23, 0xdf73, 0xffff, 0xe39d, 0x0000,
	0x5ba0, 0x9033, 0x8f9f, 0x0609, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x8db0, 0xffff, 0xffef, 0x1746,
	0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x728c, 0xffff, 0xffff, 0x3148,
	0x0000, 0x0000, 0x0000, 0x17ac, 0x5bce, 0x78cb, 0x95b7, 0xa3d2, 0xa3d2, 0xcce6, 0xffff, 0xffff, 0x3148,
	0x0000, 0x1919, 0x9bfd, 0xf86b, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0x3148,
	0x0c63, 0xcf97, 0xffff, 0xfc17, 0xb59d, 0x6981, 0x4f87, 0x3cf1, 0x3a68, 0x9276, 0xffff, 0xffff, 0x3148,
	0x8eb0, 0xffff, 0xfbf5, 0x4d33, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x7214, 0xffff, 0xffff, 0x3148,
	0xccaf, 0xffff, 0xc6ba, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x7214, 0xffff, 0xffff, 0x3148,
	0xf292, 0xffff, 0xb865, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000, 0x930c, 0xffff, 0xffff, 0x3148,
	0xc906, 0xffff, 0xf45d, 0x249f, 0x0000, 0x0000, 0x0000, 0x0000, 0x5ea0, 0xfcf1, 0xffff, 0xffff, 0x3888,
	0x6d81, 0xffff, 0xffff, 0xeaf5, 0x7dcf, 0x5533, 0x6c2b, 0xc07b, 0xfff1, 0xffff, 0xffff, 0xffff, 0x4a9d,
	0x00d4, 0xa6a1, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xd54d, 0x8399, 0xffff, 0xffff, 0x764b,
	0x0000, 0x001b, 0x4ffc, 0xbb4a, 0xe3f5, 0xeee3, 0xbd4c, 0x7e42, 0x0900, 0x1b0c, 0xb6fc, 0xb6fc, 0x7e04,
}
