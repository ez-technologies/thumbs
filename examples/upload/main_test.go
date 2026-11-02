package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func upload(t *testing.T, h http.Handler, name string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUpload_StoresTheOriginalAndItsThumbnail(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 1000; x++ {
			img.Set(x, y, color.RGBA{uint8(x), 80, uint8(y), 255})
		}
	}
	var src bytes.Buffer
	require.NoError(t, jpeg.Encode(&src, img, nil))

	rec := upload(t, uploadHandler(dir), "../../photo.jpg", src.Bytes())
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "photo.jpg", got["original"], "the client's path is dropped")
	assert.Equal(t, "photo.jpg.thumb.jpg", got["thumbnail"])

	original, err := os.ReadFile(filepath.Join(dir, "photo.jpg"))
	require.NoError(t, err)
	assert.Equal(t, src.Bytes(), original)
	f, err := os.Open(filepath.Join(dir, "photo.jpg.thumb.jpg"))
	require.NoError(t, err)
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	require.NoError(t, err)
	assert.Equal(t, [2]int{320, 160}, [2]int{cfg.Width, cfg.Height})
}

func TestUpload_RefusesANonPicture(t *testing.T) {
	dir := t.TempDir()
	rec := upload(t, uploadHandler(dir), "notes.txt", []byte("hello"))
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing is stored")
}
