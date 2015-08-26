/*
 * Copyright (c) 2013 Conformal Systems LLC.
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package fastsha256

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestSHA256(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	msg := "abc"
	digest := Sum256([]byte(msg))
	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("sha256 invalid digest")
		return
	}
}

func TestSHA256New(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	d := New()
	msg := "abc"
	d.Write([]byte(msg))
	digest := d.Sum(nil)
	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("new invalid digest %s %x", expected, got)
		return
	}
}

func TestSHA256Go(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	d := sha256.New()
	msg := "abc"
	d.Write([]byte(msg))
	digest := d.Sum(nil)
	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("append invalid digest %s %s", expected, got)
		return
	}
}

func TestSHA256Append(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	d := New()
	d.Write([]byte("a"))
	d.Write([]byte("b"))
	d.Write([]byte("c"))
	digest := d.Sum(nil)
	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("append invalid digest %s %s", expected, got)
		return
	}
}

func TestSHA256AppendGo(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	d := sha256.New()
	d.Write([]byte("a"))
	d.Write([]byte("b"))
	d.Write([]byte("c"))
	digest := d.Sum(nil)
	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("append invalid digest %s %s", expected, got)
		return
	}
}

func TestSHA256AppendAndSum(t *testing.T) {
	expected := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	d := New()
	msg := "a"
	d.Write([]byte(msg))
	digest := d.Sum(nil)

	msg = "b"
	d.Write([]byte(msg))
	digest = d.Sum(nil)

	msg = "c"
	d.Write([]byte(msg))
	digest = d.Sum(nil)

	got := fmt.Sprintf("%0x", digest)
	if got != expected {
		t.Errorf("invalid digest %x %x", expected, got)
		return
	}
}

func TestSHA256Rolling(t *testing.T) {
	var b []byte
	for i := 0; i < 20000; i++ {
		b = append(b, byte(i%256))

		d := New()
		d.Write(b)
		digest := d.Sum(nil)

		digest2 := sha256.Sum256(b)

		if string(digest) != string(digest2[:]) {
			t.Errorf("invalid digest %x %x", digest, digest2)
			return
		}
	}
}

func TestSHA256RollingDirect(t *testing.T) {
	var b []byte
	for i := 0; i < 20000; i++ {
		b = append(b, byte(i%256))

		digest := Sum256(b)
		digest2 := sha256.Sum256(b)

		if string(digest[:]) != string(digest2[:]) {
			t.Errorf("invalid digest %x %x", digest, digest2)
			return
		}
	}
}

func DoubleSha256(b []byte) []byte {
	hasher := New()
	hasher.Write(b)
	sum := hasher.Sum(nil)
	hasher.Reset()
	hasher.Write(sum)
	return hasher.Sum(nil)
}

func TestDoubleSha256(t *testing.T) {
	var b []byte
	for i := 0; i < 20000; i++ {
		b = append(b, byte(i%256))
		DoubleSha256(b)
	}
}

func TestEmpty(t *testing.T) {
	var b []byte
	DoubleSha256(b)
}

var t = strings.Repeat("a", 2049)

func BenchmarkSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sum256([]byte(t))
	}
}

func BenchmarkSha256Go(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sha256.Sum256([]byte(t))
	}
}
