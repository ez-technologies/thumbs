// Command thumbs-cli makes a JPEG thumbnail from a JPEG, PNG or WebP image.
//
//	thumbs-cli in.jpg out.jpg -max 320 -quality 80
//
// Flags may come before or after the two file names. It prints the size of the input and of the thumbnail.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"strings"

	"github.com/ez-technologies/thumbs" // also registers the JPEG, PNG and WebP decoders
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "thumbs-cli:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("thumbs-cli", flag.ContinueOnError)
	maxSide := fs.Int("max", thumbs.DefaultMaxSide, "longest side of the thumbnail, in pixels")
	quality := fs.Int("quality", thumbs.DefaultQuality, "JPEG quality, 1 to 100")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: thumbs-cli in.(jpg|png|webp) out.jpg [-max 320] [-quality 80]")
		fs.PrintDefaults()
	}

	// The flag package stops at the first file name, so parse the arguments in turns to allow flags anywhere.
	var files []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return err
		}
		args = fs.Args()
		if len(args) > 0 {
			files = append(files, args[0])
			args = args[1:]
		}
	}
	if len(files) != 2 {
		fs.Usage()
		return errors.New("need an input and an output file")
	}
	in, out := files[0], files[1]

	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	thumb, err := thumbs.MakeWith(bytes.NewReader(data), thumbs.Options{MaxSide: *maxSide, Quality: *quality})
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	if err := os.WriteFile(out, thumb, 0o644); err != nil {
		return err
	}
	tcfg, _, err := image.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s  %s %dx%d  %s\n", in, strings.ToUpper(format), cfg.Width, cfg.Height, human(len(data)))
	fmt.Fprintf(stdout, "  -> %s  JPEG %dx%d  %s  (%.1f%% of the original)\n",
		out, tcfg.Width, tcfg.Height, human(len(thumb)), 100*float64(len(thumb))/float64(len(data)))
	return nil
}

func human(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
