// Package assets loads the original Burger Time ROM set (from the user's own
// zip) and decodes the graphics and palette.
package assets

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ROMs holds the raw ROM images.
type ROMs struct {
	Main  [0x5000]byte // ab05a1,ab04,ab05,ab06,ab07 -> $b000-$ffff
	Sound [0x1000]byte // ab14
	// Gfx is the character/sprite region: three 8 KB bit planes, each made of
	// two ROMs: stored as ab12+ab13, ab10+ab11, ab8+ab9 (the last is the pixel's top bit).
	Gfx     [0x6000]byte
	BGTiles [0x1800]byte // ab00,ab01,ab02 (3 bit planes)
	BGMap   [0x0800]byte // ab03
}

var layout = []struct {
	name string
	dst  func(r *ROMs) []byte
}{
	{"ab05a1.12b", func(r *ROMs) []byte { return r.Main[0x0000:0x1000] }},
	{"ab04.9b", func(r *ROMs) []byte { return r.Main[0x1000:0x2000] }},
	{"ab06.13b", func(r *ROMs) []byte { return r.Main[0x2000:0x3000] }},
	{"ab05.10b", func(r *ROMs) []byte { return r.Main[0x3000:0x4000] }},
	{"ab07.15b", func(r *ROMs) []byte { return r.Main[0x4000:0x5000] }},
	{"ab14.12h", func(r *ROMs) []byte { return r.Sound[:] }},
	{"ab12.7k", func(r *ROMs) []byte { return r.Gfx[0x0000:0x1000] }},
	{"ab13.9k", func(r *ROMs) []byte { return r.Gfx[0x1000:0x2000] }},
	{"ab10.10k", func(r *ROMs) []byte { return r.Gfx[0x2000:0x3000] }},
	{"ab11.12k", func(r *ROMs) []byte { return r.Gfx[0x3000:0x4000] }},
	{"ab8.13k", func(r *ROMs) []byte { return r.Gfx[0x4000:0x5000] }},
	{"ab9.15k", func(r *ROMs) []byte { return r.Gfx[0x5000:0x6000] }},
	{"ab00.1b", func(r *ROMs) []byte { return r.BGTiles[0x0000:0x0800] }},
	{"ab01.3b", func(r *ROMs) []byte { return r.BGTiles[0x0800:0x1000] }},
	{"ab02.4b", func(r *ROMs) []byte { return r.BGTiles[0x1000:0x1800] }},
	{"ab03.6b", func(r *ROMs) []byte { return r.BGMap[:] }},
}

// Load reads the ROM set from a zip file, or from a directory of loose files.
func Load(path string) (*ROMs, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	r := &ROMs{}
	if st.IsDir() {
		for _, l := range layout {
			b, err := os.ReadFile(filepath.Join(path, l.name))
			if err != nil {
				return nil, err
			}
			if err := put(r, l.name, l.dst(r), b); err != nil {
				return nil, err
			}
		}
		return r, nil
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return loadZip(r, &zr.Reader, path)
}

func put(r *ROMs, name string, d, b []byte) error {
	if len(b) != len(d) {
		return fmt.Errorf("%s: size %d, want %d", name, len(b), len(d))
	}
	copy(d, b)
	return nil
}

// Embedded is a ROM zip compiled into the program (see embed.go, build tag
// "embedrom"); nil in the normal build.
var Embedded []byte

// LoadEmbedded reads the ROM set from the compiled-in zip.
func LoadEmbedded() (*ROMs, error) {
	zr, err := zip.NewReader(bytes.NewReader(Embedded), int64(len(Embedded)))
	if err != nil {
		return nil, err
	}
	return loadZip(&ROMs{}, zr, "embedded ROM")
}

func loadZip(r *ROMs, zr *zip.Reader, src string) (*ROMs, error) {
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[filepath.Base(f.Name)] = f
	}
	for _, l := range layout {
		f, ok := files[l.name]
		if !ok {
			return nil, fmt.Errorf("%s: missing ROM file %s", src, l.name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		if err := put(r, l.name, l.dst(r), b); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// MissingError reports ROM files that none of the supplied sets contain.
type MissingError struct {
	Missing []string
	// DataEast is true when the sets look like the Data East parent set
	// (aa04.9b ...), whose program ROMs differ from the Midway set this
	// program is built from.
	DataEast bool
	Sources  []string
}

func (e *MissingError) Error() string {
	return fmt.Sprintf("missing ROM files: %v (looked in %v)", e.Missing, e.Sources)
}

// LoadMulti reads the ROM set from one or more zips or folders and merges them
// by file name (first one wins), so split, merged and full MAME sets all work.
func LoadMulti(paths []string) (*ROMs, error) {
	files := map[string][]byte{}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.IsDir() {
			ents, _ := os.ReadDir(p)
			for _, e := range ents {
				if b, err := os.ReadFile(filepath.Join(p, e.Name())); err == nil {
					if _, ok := files[e.Name()]; !ok {
						files[e.Name()] = b
					}
				}
			}
			continue
		}
		zr, err := zip.OpenReader(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		for _, f := range zr.File {
			name := filepath.Base(f.Name)
			if _, ok := files[name]; ok {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			files[name] = b
		}
		zr.Close()
	}
	r := &ROMs{}
	var missing []string
	for _, l := range layout {
		b, ok := files[l.name]
		if !ok {
			missing = append(missing, l.name)
			continue
		}
		if err := put(r, l.name, l.dst(r), b); err != nil {
			return nil, err
		}
	}
	if len(missing) > 0 {
		_, de := files["aa04.9b"]
		return nil, &MissingError{Missing: missing, DataEast: de, Sources: paths}
	}
	return r, nil
}
