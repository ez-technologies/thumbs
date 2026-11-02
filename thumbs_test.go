package thumbs

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func solid(w, h int, left, right color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.Set(x, y, left)
			} else {
				img.Set(x, y, right)
			}
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}))
	return b.Bytes()
}

func size(t *testing.T, data []byte) (int, int) {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	return img.Bounds().Dx(), img.Bounds().Dy()
}

var red, blue = color.RGBA{220, 20, 20, 255}, color.RGBA{20, 20, 220, 255}

func TestThumbs_ShrinksToTheLongestSideAndKeepsTheShape(t *testing.T) {
	wide, err := Make(bytes.NewReader(encodeJPEG(t, solid(1600, 900, red, blue))))
	require.NoError(t, err)
	w, h := size(t, wide)
	assert.Equal(t, [2]int{320, 180}, [2]int{w, h})

	var p bytes.Buffer
	require.NoError(t, png.Encode(&p, solid(600, 1200, red, blue)))
	tall, err := Make(bytes.NewReader(p.Bytes()))
	require.NoError(t, err)
	w, h = size(t, tall)
	assert.Equal(t, [2]int{160, 320}, [2]int{w, h})

	small, err := Make(bytes.NewReader(encodeJPEG(t, solid(100, 50, red, blue))))
	require.NoError(t, err)
	w, h = size(t, small)
	assert.Equal(t, [2]int{100, 50}, [2]int{w, h}, "a small image is not enlarged")

	assert.Less(t, len(wide), 40_000, "a thumbnail is small")
}

func TestThumbs_OptionsSetTheSize(t *testing.T) {
	out, err := MakeWith(bytes.NewReader(encodeJPEG(t, solid(1600, 900, red, blue))), Options{MaxSide: 64, Quality: 50})
	require.NoError(t, err)
	w, h := size(t, out)
	assert.Equal(t, [2]int{64, 36}, [2]int{w, h})
}

func TestThumbs_ANonPictureIsRefused(t *testing.T) {
	_, err := Make(bytes.NewReader([]byte("not a picture")))
	assert.Error(t, err)
}

// withOrientation puts an EXIF block saying the given orientation right after a JPEG's start.
func withOrientation(t *testing.T, data []byte, o uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	tiff.WriteString("II*\x00")
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(8))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(1))
	_ = binary.Write(&tiff, binary.LittleEndian, []uint16{0x0112, 3})
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(1))
	_ = binary.Write(&tiff, binary.LittleEndian, []uint16{o, 0})
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(0))
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	out := append([]byte{}, data[:2]...)
	out = append(out, seg...)
	out = append(out, payload...)
	return append(out, data[2:]...)
}

func TestThumbs_TurnsThePictureTheWayTheCameraMeant(t *testing.T) {
	// Red on the left, blue on the right, saved lying on its side with "turn a quarter clockwise" (6): standing
	// up, it is tall, red on top.
	data := withOrientation(t, encodeJPEG(t, solid(400, 200, red, blue)), 6)
	assert.Equal(t, 6, exifOrientation(data))
	out, err := Make(bytes.NewReader(data))
	require.NoError(t, err)
	img, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, [2]int{160, 320}, [2]int{img.Bounds().Dx(), img.Bounds().Dy()})
	r, _, b, _ := img.At(80, 40).RGBA()
	assert.Greater(t, r, b, "the top is the red half")
	r, _, b, _ = img.At(80, 280).RGBA()
	assert.Greater(t, b, r, "the bottom is the blue half")

	assert.Equal(t, 1, exifOrientation(encodeJPEG(t, solid(10, 10, red, blue))), "no EXIF: as it is")
}
