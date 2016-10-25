package main

import (
	"encoding/hex"
	"flag"
	"io"
	"log"
	"math"
	"os"
)

var (
	decode  bool
	lineLen = flag.Int("n", math.MaxInt32, "max encoded output line `length`")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("hex: ")
	flag.BoolVar(&decode, "d", false, "decode (negates -e)")
	flagNotBoolVar(&decode, "e", true, "encode (negates -d) (default true)")
	flag.Parse()
	var r io.Reader = &encodeReader{os.Stdin}
	if *lineLen >= 2 {
		r = &splitLineReader{r: r, max: *lineLen - *lineLen%2}
	}
	if decode {
		r = &decodeReader{r: &stripSpaceReader{os.Stdin}}
	}
	_, err := io.Copy(os.Stdout, r)
	if err != nil {
		log.Fatal(err)
	}
}

type encodeReader struct {
	r io.Reader
}

func (r *encodeReader) Read(p []byte) (int, error) {
	if len(p)%2 == 1 {
		p = p[:len(p)-1]
	}
	h := p[len(p)/2:]
	n, err := r.r.Read(h)
	return hex.Encode(p, h[:n]), err
}

type splitLineReader struct {
	r   io.Reader
	n   int
	max int
}

func (r *splitLineReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	p = p[:len(p)-1] // leave room for possible '\n' below
	if len(p) > r.max-r.n {
		p = p[:r.max-r.n]
	}
	n, err := r.r.Read(p)
	p = p[:n]
	r.n += n
	if r.n >= r.max || err == io.EOF {
		p = p[:n+1]
		p[n] = '\n'
		r.n = 0
	}
	return len(p), err
}

type decodeReader struct {
	r   io.Reader
	par int  // parity of last read (0 or 1)
	b   byte // odd byte from last read (if any)
}

func (r *decodeReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.par == 1 {
		p[0] = r.b
	}
	n, err := r.r.Read(p[r.par:])
	p = p[:r.par+n]
	r.par = len(p) % 2
	if r.par == 1 {
		r.b = p[len(p)-1]
		p = p[:len(p)-1]
	}
	d, err1 := hex.Decode(p, p)
	if err1 != nil {
		return d, err1
	}
	if r.par == 1 && err == io.EOF {
		err = hex.ErrLength
	}
	return d, err
}

type stripSpaceReader struct {
	r io.Reader
}

func (r *stripSpaceReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	w := 0
	for _, b := range p[:n] {
		if !isSpace(b) {
			p[w] = b
			w++
		}
	}
	return w, err
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
