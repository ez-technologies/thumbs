package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	require.NoError(t, os.WriteFile(path, b.Bytes(), 0o644))
}

func TestCLI_FlagsAfterTheFileNames(t *testing.T) {
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in.png"), filepath.Join(dir, "out.jpg")
	writePNG(t, in, 800, 400)

	var stdout bytes.Buffer
	require.NoError(t, run([]string{in, out, "-max", "100", "-quality", "70"}, &stdout))

	f, err := os.Open(out)
	require.NoError(t, err)
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Equal(t, [2]int{100, 50}, [2]int{cfg.Width, cfg.Height})
	assert.True(t, strings.Contains(stdout.String(), "PNG 800x400"), stdout.String())
	assert.True(t, strings.Contains(stdout.String(), "JPEG 100x50"), stdout.String())
}

func TestCLI_NeedsTwoFiles(t *testing.T) {
	var stdout bytes.Buffer
	assert.Error(t, run([]string{"only-one.png"}, &stdout))
}
