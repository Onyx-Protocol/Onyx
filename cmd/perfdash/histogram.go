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

// TODO(kr): collect more fine-grain stats

const (
	// Use these to go to p99.99999
	// pixel0Quantile = 0.5
	// decayPerPixel  = 0.98
	pixel0Quantile = 0.4
	decayPerPixel  = 0.99
)

var (
	dims = image.Rect(0, 0, 800, 100)

	// In reverse order, newest to oldest,
	// so strongest to faintest color.
	// Don't forget, color.RGBA fields are
	// alpha premultiplied.
	lineColors = [...]color.Color{
		color.RGBA{B: 0xff, A: 0xff},
		color.RGBA{B: 0xbb, A: 0xbb},
		color.RGBA{B: 0x66, A: 0x66},
		color.RGBA{B: 0x22, A: 0x22},
	}
	curLineColor = color.RGBA{R: 0x55, B: 0x55, A: 0x55}
	labelColor   = color.Black
	drawer       = font.Drawer{
		Src:  image.NewUniform(labelColor),
		Face: inconsolata.Regular8x16,
	}
)

func histogram(w http.ResponseWriter, req *http.Request) {
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

	img := newImage(dims)
	d := drawer
	d.Dst = img

	var (
		hists   []*hdrhistogram.Histogram
		trueMax time.Duration
		over    int
		total   int64
	)
	for _, b := range latency.Buckets {
		hists = append(hists, hdrhistogram.Import(&b.Histogram))
		over += b.Over
		total += int64(b.Over)
		if d := time.Duration(b.Max); d > trueMax {
			trueMax = d
		}
	}

	var max int64
	for _, hist := range hists {
		if v := hist.Max(); v > max {
			max = v
		}
		total += hist.TotalCount()
	}

	histMax := "no histogram data"
	if max > 0 {
		max += max / 10
		max = roundms(max)
		if max < int64(10*time.Millisecond) {
			max += int64(time.Millisecond)
		}
		complete := hists[:len(hists)-1]
		for i, hist := range complete {
			rindex := len(complete) - i - 1
			graph(img, hist, max, lineColors[rindex%len(lineColors)])
		}
		// special color for incomplete bucket
		graph(img, hists[len(hists)-1], max, curLineColor)
		label(img, labelColor)
		histMax = time.Duration(max).String()
	}
	drawf(d, 4, 20, "%s (%v max)", histMax, trueMax)
	drawf(d, 4, 38, "%s", name)
	drawf(d, 4, 54, "%d events (%d over)", total, over)
	png.Encode(w, img)
}

func graph(img *image.RGBA, hist *hdrhistogram.Histogram, ymax int64, color color.Color) {
	graph := img.SubImage(img.Bounds().Inset(2)).(*image.RGBA)
	gdims := graph.Bounds()
	d := drawer
	d.Dst = graph

	labelPixels := 50
	prevY := int(scale(valueAtPixel(hist, 0), ymax, int64(gdims.Dy())))
	for x := 1; x < gdims.Dx(); x++ {
		v := valueAtPixel(hist, x)
		y := int(scale(v, ymax, int64(gdims.Dy())))
		vLineSeg(graph, gdims.Min.X+x, gdims.Max.Y-y-1, gdims.Max.Y-prevY-1, color)
		labelPixels++
		prevY = y
	}
}

func vLineSeg(img *image.RGBA, x, y0, y1 int, color color.Color) {
	dims := image.Rect(x, y0, x+1, y1+1)
	draw.Draw(img, dims, image.NewUniform(color), image.ZP, draw.Over)
}

func label(img *image.RGBA, color color.Color) {
	graph := img.SubImage(img.Bounds().Inset(2)).(*image.RGBA)
	gdims := graph.Bounds()
	d := drawer
	d.Dst = graph

	labelDigits := 0
	labelPixels := 0
	for x := 0; x < gdims.Dx(); x++ {
		q := quantileAtPixel(x)
		if dig := digits(1 - q); labelPixels >= 50 && dig > labelDigits {
			labelPixels = 0
			labelDigits = dig
			for i := 0; i < 5; i++ {
				graph.Set(gdims.Min.X+x, gdims.Max.Y-i-1, color)
			}
			prec := labelDigits - 3
			if prec < 0 {
				prec = 0
			}
			drawf(d, gdims.Min.X+x+2, gdims.Max.Y-2, "p%.*f", prec, 100*q)
		}
		labelPixels++
	}

	// special case for first pixel
	drawf(d, 4, gdims.Max.Y-2, "p%.*f", 0, 100*pixel0Quantile)
}

func quantileAtPixel(n int) float64 {
	return 1 - (1-pixel0Quantile)*math.Pow(decayPerPixel, float64(n))
}

func valueAtPixel(hist *hdrhistogram.Histogram, n int) int64 {
	return hist.ValueAtQuantile(100 * quantileAtPixel(n))
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
