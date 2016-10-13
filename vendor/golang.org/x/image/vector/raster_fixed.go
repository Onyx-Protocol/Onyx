// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

// This file contains a fixed point math implementation of the vector
// graphics rasterizer.

import (
	"golang.org/x/image/math/f32"
)

const (
	// ϕ is the number of binary digits after the fixed point.
	//
	// For example, if ϕ == 10 (and int1ϕ is based on the int32 type) then we
	// are using 22.10 fixed point math.
	//
	// When changing this number, also change the assembly code (search for ϕ
	// in the .s files).
	ϕ = 10

	fxOne          int1ϕ = 1 << ϕ
	fxOneAndAHalf  int1ϕ = 1<<ϕ + 1<<(ϕ-1)
	fxOneMinusIota int1ϕ = 1<<ϕ - 1 // Used for rounding up.
)

// int1ϕ is a signed fixed-point number with 1*ϕ binary digits after the fixed
// point.
type int1ϕ int32

// int2ϕ is a signed fixed-point number with 2*ϕ binary digits after the fixed
// point.
//
// The Rasterizer's bufU32 field, nominally of type []uint32 (since that slice
// is also used by other code), can be thought of as a []int2ϕ during the
// fixedLineTo method. Lines of code that are actually like:
//	buf[i] += uint32(etc) // buf has type []uint32.
// can be thought of as
//	buf[i] += int2ϕ(etc)  // buf has type []int2ϕ.
type int2ϕ int32

func fixedMax(x, y int1ϕ) int1ϕ {
	if x > y {
		return x
	}
	return y
}

func fixedMin(x, y int1ϕ) int1ϕ {
	if x < y {
		return x
	}
	return y
}

func fixedFloor(x int1ϕ) int32 { return int32(x >> ϕ) }
func fixedCeil(x int1ϕ) int32  { return int32((x + fxOneMinusIota) >> ϕ) }

func (z *Rasterizer) fixedLineTo(b f32.Vec2) {
	a := z.pen
	z.pen = b
	dir := int1ϕ(1)
	if a[1] > b[1] {
		dir, a, b = -1, b, a
	}
	// Horizontal line segments yield no change in coverage. Almost horizontal
	// segments would yield some change, in ideal math, but the computation
	// further below, involving 1 / (b[1] - a[1]), is unstable in fixed point
	// math, so we treat the segment as if it was perfectly horizontal.
	if b[1]-a[1] <= 0.000001 {
		return
	}
	dxdy := (b[0] - a[0]) / (b[1] - a[1])

	ay := int1ϕ(a[1] * float32(fxOne))
	by := int1ϕ(b[1] * float32(fxOne))

	x := int1ϕ(a[0] * float32(fxOne))
	y := fixedFloor(ay)
	yMax := fixedCeil(by)
	if yMax > int32(z.size.Y) {
		yMax = int32(z.size.Y)
	}
	width := int32(z.size.X)

	for ; y < yMax; y++ {
		dy := fixedMin(int1ϕ(y+1)<<ϕ, by) - fixedMax(int1ϕ(y)<<ϕ, ay)
		xNext := x + int1ϕ(float32(dy)*dxdy)
		if y < 0 {
			x = xNext
			continue
		}
		buf := z.bufU32[y*width:]
		d := dy * dir
		x0, x1 := x, xNext
		if x > xNext {
			x0, x1 = x1, x0
		}
		x0i := fixedFloor(x0)
		x0Floor := int1ϕ(x0i) << ϕ
		x1i := fixedCeil(x1)
		x1Ceil := int1ϕ(x1i) << ϕ

		if x1i <= x0i+1 {
			xmf := (x+xNext)>>1 - x0Floor
			if i := clamp(x0i+0, width); i < uint(len(buf)) {
				buf[i] += uint32(d * (fxOne - xmf))
			}
			if i := clamp(x0i+1, width); i < uint(len(buf)) {
				buf[i] += uint32(d * xmf)
			}
		} else {
			oneOverS := x1 - x0
			twoOverS := 2 * oneOverS
			x0f := x0 - x0Floor
			oneMinusX0f := fxOne - x0f
			oneMinusX0fSquared := oneMinusX0f * oneMinusX0f
			x1f := x1 - x1Ceil + fxOne
			x1fSquared := x1f * x1f

			// These next two variables are unused, as rounding errors are
			// minimized when we delay the division by oneOverS for as long as
			// possible. These lines of code (and the "In ideal math" comments
			// below) are commented out instead of deleted in order to aid the
			// comparison with the floating point version of the rasterizer.
			//
			// a0 := ((oneMinusX0f * oneMinusX0f) >> 1) / oneOverS
			// am := ((x1f * x1f) >> 1) / oneOverS

			if i := clamp(x0i, width); i < uint(len(buf)) {
				// In ideal math: buf[i] += uint32(d * a0)
				D := oneMinusX0fSquared
				D *= d
				D /= twoOverS
				buf[i] += uint32(D)
			}

			if x1i == x0i+2 {
				if i := clamp(x0i+1, width); i < uint(len(buf)) {
					// In ideal math: buf[i] += uint32(d * (fxOne - a0 - am))
					D := twoOverS<<ϕ - oneMinusX0fSquared - x1fSquared
					D *= d
					D /= twoOverS
					buf[i] += uint32(D)
				}
			} else {
				// This is commented out for the same reason as a0 and am.
				//
				// a1 := ((fxOneAndAHalf - x0f) << ϕ) / oneOverS

				if i := clamp(x0i+1, width); i < uint(len(buf)) {
					// In ideal math: buf[i] += uint32(d * (a1 - a0))
					//
					// Convert to int64 to avoid overflow. Without that,
					// TestRasterizePolygon fails.
					D := int64((fxOneAndAHalf-x0f)<<(ϕ+1) - oneMinusX0fSquared)
					D *= int64(d)
					D /= int64(twoOverS)
					buf[i] += uint32(D)
				}
				dTimesS := uint32((d << (2 * ϕ)) / oneOverS)
				for xi := x0i + 2; xi < x1i-1; xi++ {
					if i := clamp(xi, width); i < uint(len(buf)) {
						buf[i] += dTimesS
					}
				}

				// This is commented out for the same reason as a0 and am.
				//
				// a2 := a1 + (int1ϕ(x1i-x0i-3)<<(2*ϕ))/oneOverS

				if i := clamp(x1i-1, width); i < uint(len(buf)) {
					// In ideal math: buf[i] += uint32(d * (fxOne - a2 - am))
					//
					// Convert to int64 to avoid overflow. Without that,
					// TestRasterizePolygon fails.
					D := int64(twoOverS << ϕ)
					D -= int64((fxOneAndAHalf - x0f) << (ϕ + 1))
					D -= int64((x1i - x0i - 3) << (2*ϕ + 1))
					D -= int64(x1fSquared)
					D *= int64(d)
					D /= int64(twoOverS)
					buf[i] += uint32(D)
				}
			}

			if i := clamp(x1i, width); i < uint(len(buf)) {
				// In ideal math: buf[i] += uint32(d * am)
				D := x1fSquared
				D *= d
				D /= twoOverS
				buf[i] += uint32(D)
			}
		}

		x = xNext
	}
}

func fixedAccumulateOpOver(dst []uint8, src []uint32) {
	acc := int2ϕ(0)
	for i, v := range src {
		acc += int2ϕ(v)
		a := acc
		if a < 0 {
			a = -a
		}
		a >>= 2*ϕ - 16
		if a > 0xffff {
			a = 0xffff
		}
		// This algorithm comes from the standard library's image/draw package.
		dstA := uint32(dst[i]) * 0x101
		maskA := uint32(a)
		outA := dstA*(0xffff-maskA)/0xffff + maskA
		dst[i] = uint8(outA >> 8)
	}
}

func fixedAccumulateOpSrc(dst []uint8, src []uint32) {
	// Sanity check that len(dst) >= len(src).
	if len(dst) < len(src) {
		return
	}

	acc := int2ϕ(0)
	for i, v := range src {
		acc += int2ϕ(v)
		a := acc
		if a < 0 {
			a = -a
		}
		a >>= 2*ϕ - 8
		if a > 0xff {
			a = 0xff
		}
		dst[i] = uint8(a)
	}
}

func fixedAccumulateMask(buf []uint32) {
	acc := int2ϕ(0)
	for i, v := range buf {
		acc += int2ϕ(v)
		a := acc
		if a < 0 {
			a = -a
		}
		a >>= 2*ϕ - 16
		if a > 0xffff {
			a = 0xffff
		}
		buf[i] = uint32(a)
	}
}
