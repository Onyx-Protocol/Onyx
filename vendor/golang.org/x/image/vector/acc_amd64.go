// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !appengine
// +build gc
// +build !noasm

package vector

func haveSSE4_1() bool

var haveFixedAccumulateSIMD = haveSSE4_1()

const haveFloatingAccumulateSIMD = true

//go:noescape
func fixedAccumulateOpSrcSIMD(dst []uint8, src []uint32)

//go:noescape
func floatingAccumulateOpSrcSIMD(dst []uint8, src []float32)
