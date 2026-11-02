// Package thumbs makes JPEG thumbnails: an image (JPEG, PNG or WebP) is scaled down to at most a given number of
// pixels on its longest side, turned upright by its EXIF orientation, and encoded as JPEG.
package thumbs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // decoded by image.Decode
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // decoded by image.Decode
)

// DefaultMaxSide is the default longest side of a thumbnail, in pixels.
const DefaultMaxSide = 320

// DefaultQuality is the default JPEG quality of a thumbnail (1 to 100).
const DefaultQuality = 80

// Suffix is a conventional name suffix for storing a thumbnail next to its original: "<key>" and "<key>.thumb.jpg".
const Suffix = ".thumb.jpg"

// Options control how a thumbnail is made. Zero values mean the defaults.
type Options struct {
	MaxSide int // longest side in pixels; 0 means DefaultMaxSide
	Quality int // JPEG quality 1 to 100; 0 means DefaultQuality
}

// Make reads an image (JPEG, PNG or WebP) and returns its thumbnail as JPEG, using the default options.
func Make(r io.Reader) ([]byte, error) {
	return MakeWith(r, Options{})
}

// MakeWith reads an image (JPEG, PNG or WebP) and returns its thumbnail as JPEG. An image already small enough is
// not enlarged, only re-encoded. Transparent areas are drawn on white.
func MakeWith(r io.Reader, opts Options) ([]byte, error) {
	maxSide := opts.MaxSide
	if maxSide <= 0 {
		maxSide = DefaultMaxSide
	}
	quality := opts.Quality
	if quality <= 0 {
		quality = DefaultQuality
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	src = orient(src, exifOrientation(data))
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, errors.New("thumbs: an empty picture")
	}
	if w > maxSide || h > maxSide {
		if w >= h {
			w, h = maxSide, max(1, h*maxSide/w)
		} else {
			w, h = max(1, w*maxSide/h), maxSide
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// exifOrientation reads the EXIF orientation of a JPEG (1 to 8), or 1 when there is none or it cannot be read.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD9 || marker == 0xDA { // end, or the picture itself: no EXIF before it
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(t[4:8]))
	if ifd+2 > len(t) {
		return 1
	}
	n := int(order.Uint16(t[ifd : ifd+2]))
	for e := 0; e < n; e++ {
		at := ifd + 2 + e*12
		if at+12 > len(t) {
			return 1
		}
		if order.Uint16(t[at:at+2]) == 0x0112 {
			if v := int(order.Uint16(t[at+8 : at+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// orient turns a picture by its EXIF orientation, so it stands as the camera meant.
func orient(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2: // mirrored
				dx, dy = w-1-x, y
			case 3: // upside down
				dx, dy = w-1-x, h-1-y
			case 4: // mirrored upside down
				dx, dy = x, h-1-y
			case 5: // mirrored, turned
				dx, dy = y, x
			case 6: // turned a quarter clockwise
				dx, dy = h-1-y, x
			case 7: // mirrored, turned the other way
				dx, dy = h-1-y, w-1-x
			case 8: // turned a quarter anticlockwise
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
