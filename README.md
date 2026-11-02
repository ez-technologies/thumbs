# thumbs

A small Go package that makes JPEG thumbnails. It reads a JPEG, PNG or WebP image, turns it upright by its JPEG EXIF
orientation, scales it down to at most a given longest side (320 pixels by default), draws transparent areas on white
and encodes the result as JPEG. Images that are already small are not enlarged.

## Install

```sh
go get github.com/ez-technologies/thumbs
```

## Usage

```go
f, err := os.Open("photo.jpg")
if err != nil {
	return err
}
defer f.Close()

thumb, err := thumbs.Make(f) // 320 px longest side, JPEG quality 80
if err != nil {
	return err
}

// Or with your own size and quality:
// thumb, err := thumbs.MakeWith(f, thumbs.Options{MaxSide: 200, Quality: 85})

return os.WriteFile("photo"+thumbs.Suffix, thumb, 0o644)
```

## Licence

MIT, see [LICENSE](LICENSE).
