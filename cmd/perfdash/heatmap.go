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
	"time"

	"github.com/codahale/hdrhistogram"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

// TODO(kr): display more than one bucket (in time-series?)
// TODO(kr): collect more fine-grain stats
// TODO(kr): coalesce /debug/vars requests (cache them?)

var (
	dims       = image.Rect(0, 0, 800, 100)
	lineColor  = color.RGBA{R: 0xff, A: 0xff}
	labelColor = color.Black
	drawer     = font.Drawer{
		Src:  image.NewUniform(labelColor),
		Face: inconsolata.Regular8x16,
	}

	xLabels = map[int]bool{
		0:   true,
		100: true,
		200: true,
		300: true,
		400: true,
		500: true,
		600: true,
		700: true,
		790: true,
	}
)

func heatmap(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	baseURL := query.Get("baseurl")
	name := query.Get("name")

	var debugvars struct {
		Latency map[string]struct {
			Buckets []struct {
				Over      int
				Histogram hdrhistogram.Snapshot
			}
		}
	}
	err := getDebugVars(baseURL, &debugvars)
	if err != nil {
		pngError(w, err)
		return
	}

	latency, ok := debugvars.Latency[name]
	if !ok {
		pngError(w, errors.New("not found"))
		return
	}

	b0 := latency.Buckets[0]
	img := newImage(dims)
	d := drawer
	d.Dst = img
	drawf(d, 4, 20, "%v", time.Duration(b0.Histogram.HighestTrackableValue))
	drawf(d, 4, 38, "%s", name)
	drawf(d, 4, 54, "over: %d", b0.Over)
	graph(img, hdrhistogram.Import(&b0.Histogram), lineColor, labelColor)
	png.Encode(w, img)
}

func graph(img *image.RGBA, hist *hdrhistogram.Histogram, lineColor, labelColor color.Color) {
	graph := img.SubImage(img.Bounds().Inset(1)).(*image.RGBA)
	gdims := graph.Bounds().Inset(1)
	d := drawer
	d.Dst = graph

	labelDigits := 0
	labelPixels := 50
	p := 0.5
	for x := 0; x < gdims.Max.X; x++ {
		q := 1 - p
		v := hist.ValueAtQuantile(100 * q)
		y := int(scale(v, hist.HighestTrackableValue(), int64(gdims.Max.Y)))
		graph.Set(x, gdims.Max.Y-y, lineColor)
		if labelPixels >= 50 && digits(p) > labelDigits {
			labelPixels = 0
			labelDigits = digits(p)
			for i := 0; i < 5; i++ {
				graph.Set(x, gdims.Max.Y-i, labelColor)
			}
			prec := labelDigits - 3
			if prec < 0 {
				prec = 0
			}
			drawf(d, x+4, gdims.Max.Y-2, "p%.*f", prec, 100*q)
		}
		labelPixels++
		p *= 0.98
	}

	// special case for p50
	drawf(d, 4, gdims.Max.Y-2, "p50")
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
