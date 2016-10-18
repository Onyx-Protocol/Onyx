package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/codahale/hdrhistogram"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

// TODO(kr): display more than one bucket (in time-series?)
// TODO(kr): collect more fine-grain stats

var (
	dims       = image.Rect(0, 0, 800, 100)
	lineColor  = color.RGBA{R: 0xff, A: 0xff}
	labelColor = color.Black
	drawer     = font.Drawer{
		Src:  image.NewUniform(labelColor),
		Face: inconsolata.Regular8x16,
	}
)

func heatmap(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	name := query.Get("name")
	id, err := strconv.Atoi(query.Get("id"))
	if err != nil {
		pngError(w, err)
		return
	}

	dv := getDebugVars(id)
	if dv == nil {
		pngError(w, errors.New("not found"))
		return
	}

	latency, ok := dv.Latency[name]
	if !ok {
		pngError(w, errors.New("not found"))
		return
	}

	b0 := latency.Buckets[0]
	img := newImage(dims)
	d := drawer
	d.Dst = img
	hist := hdrhistogram.Import(&b0.Histogram)

	_, max := valueAtPixel(hist, img.Bounds().Inset(2).Dx())
	drawf(d, 4, 38, "%s", name)
	drawf(d, 4, 54, "over: %d", b0.Over)
	if max == 0 {
		drawf(d, 4, 20, "no histogram data")
	} else {
		max += max / 10
		max = roundms(max)
		if max < int64(10*time.Millisecond) {
			max += int64(time.Millisecond)
		}
		drawf(d, 4, 20, "%v", time.Duration(max))
		graph(img, hist, max, lineColor, labelColor)
	}
	png.Encode(w, img)
}

func graph(img *image.RGBA, hist *hdrhistogram.Histogram, ymax int64, lineColor, labelColor color.Color) {
	graph := img.SubImage(img.Bounds().Inset(2)).(*image.RGBA)
	gdims := graph.Bounds()
	d := drawer
	d.Dst = graph

	labelDigits := 0
	labelPixels := 50
	for x := 0; x < gdims.Dx(); x++ {
		q, v := valueAtPixel(hist, x)
		y := int(scale(v, ymax, int64(gdims.Dy())))
		graph.Set(gdims.Min.X+x, gdims.Max.Y-y-1, lineColor)
		if dig := digits(1 - q); labelPixels >= 50 && dig > labelDigits {
			labelPixels = 0
			labelDigits = dig
			for i := 0; i < 5; i++ {
				graph.Set(gdims.Min.X+x, gdims.Max.Y-i-1, labelColor)
			}
			prec := labelDigits - 3
			if prec < 0 {
				prec = 0
			}
			drawf(d, gdims.Min.X+x+2, gdims.Max.Y-2, "p%.*f", prec, 100*q)
		}
		labelPixels++
	}

	// special case for p50
	drawf(d, 4, gdims.Max.Y-2, "p50")
}

func valueAtPixel(hist *hdrhistogram.Histogram, n int) (quantile float64, v int64) {
	const pixel0, decayPerPixel = 0.5, 0.98
	q := 1 - pixel0*math.Pow(decayPerPixel, float64(n))
	return q, hist.ValueAtQuantile(100 * q)
}

func roundms(n int64) int64 {
	return int64(time.Duration(n) / time.Millisecond * time.Millisecond)
}

func digits(p float64) int {
	return -int(math.Floor(math.Log10(p)))
}

func drawf(d font.Drawer, x, y int, format string, args ...interface{}) {
	d.Dot = fixed.P(x, y)
	d.DrawString(fmt.Sprintf(format, args...))
}

func scale(v, from, to int64) int64 {
	if v > from {
		return to
	} else if v < 0 {
		return 0
	}
	return v * to / from
}

func pngError(w http.ResponseWriter, err error) {
	img := newImage(dims)
	d := drawer
	d.Dst = img
	drawf(d, 10, 25, "%s", err.Error())
	w.WriteHeader(500)
	png.Encode(w, img)
}

func newImage(dims image.Rectangle) *image.RGBA {
	img := image.NewRGBA(dims)
	draw.Draw(img, dims, image.Black, image.ZP, draw.Over)
	draw.Draw(img, dims.Inset(1), image.White, image.ZP, draw.Over)
	return img
}
