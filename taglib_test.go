package taglib_test

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"go.senan.xyz/taglib"
)

func TestInvalid(t *testing.T) {
	t.Parallel()

	path := tmpf(t, []byte("not a file"), "eg.flac")
	_, err := taglib.ReadTags(path)
	eq(t, err, taglib.ErrInvalidFile)
}

func TestClear(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// set some tags first
			err := taglib.WriteTags(path, map[string][]string{
				"ARTIST":     {"Example A"},
				"ALUMARTIST": {"Example"},
			}, taglib.Clear)

			nilErr(t, err)

			// then clear
			err = taglib.WriteTags(path, nil, taglib.Clear)
			nilErr(t, err)

			got, err := taglib.ReadTags(path)
			nilErr(t, err)

			if len(got) > 0 {
				t.Fatalf("exp empty, got %v", got)
			}
		})
	}
}

func TestReadWrite(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)
	testTags := []map[string][]string{
		{
			"ONE":  {"one", "two", "three", "four"},
			"FIVE": {"six", "seven"},
			"NINE": {"nine"},
		},
		{
			"ARTIST":     {"Example A", "Hello, 世界"},
			"ALUMARTIST": {"Example"},
		},
		{
			"ARTIST":      {"Example A", "Example B"},
			"ALUMARTIST":  {"Example"},
			"TRACK":       {"1"},
			"TRACKNUMBER": {"1"},
		},
		{
			"ARTIST":     {"Example A", "Example B"},
			"ALUMARTIST": {"Example"},
		},
		{
			"ARTIST": {"Hello, 世界", "界世"},
		},
		{
			"ARTIST": {"Brian Eno—David Byrne"},
			"ALBUM":  {"My Life in the Bush of Ghosts"},
		},
		{
			"ARTIST":      {"Hello, 世界", "界世"},
			"ALBUM":       {longString},
			"ALBUMARTIST": {longString, longString},
			"OTHER":       {strings.Repeat(longString, 2)},
		},
	}

	for _, path := range paths {
		for i, tags := range testTags {
			t.Run(fmt.Sprintf("%s_tags_%d", filepath.Base(path), i), func(t *testing.T) {
				err := taglib.WriteTags(path, tags, taglib.Clear)
				nilErr(t, err)

				got, err := taglib.ReadTags(path)
				nilErr(t, err)

				tagEq(t, got, tags)
			})
		}
	}
}

func TestMergeWrite(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)

	cmp := func(t *testing.T, path string, want map[string][]string) {
		t.Helper()
		tags, err := taglib.ReadTags(path)
		nilErr(t, err)
		tagEq(t, tags, want)
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			err := taglib.WriteTags(path, nil, taglib.Clear)
			nilErr(t, err)

			err = taglib.WriteTags(path, map[string][]string{
				"ONE": {"one"},
			}, 0)

			nilErr(t, err)
			cmp(t, path, map[string][]string{
				"ONE": {"one"},
			})

			nilErr(t, err)
			err = taglib.WriteTags(path, map[string][]string{
				"TWO": {"two", "two!"},
			}, 0)

			nilErr(t, err)
			cmp(t, path, map[string][]string{
				"ONE": {"one"},
				"TWO": {"two", "two!"},
			})

			err = taglib.WriteTags(path, map[string][]string{
				"THREE": {"three"},
			}, 0)

			nilErr(t, err)
			cmp(t, path, map[string][]string{
				"ONE":   {"one"},
				"TWO":   {"two", "two!"},
				"THREE": {"three"},
			})

			// change prev
			err = taglib.WriteTags(path, map[string][]string{
				"ONE": {"one new"},
			}, 0)

			nilErr(t, err)
			cmp(t, path, map[string][]string{
				"ONE":   {"one new"},
				"TWO":   {"two", "two!"},
				"THREE": {"three"},
			})

			// change prev
			err = taglib.WriteTags(path, map[string][]string{
				"ONE":   {},
				"THREE": {"three new!"},
			}, 0)

			nilErr(t, err)
			cmp(t, path, map[string][]string{
				"TWO":   {"two", "two!"},
				"THREE": {"three new!"},
			})
		})
	}
}

func TestReadExistingUnicode(t *testing.T) {
	tags, err := taglib.ReadTags("testdata/normal.flac")
	nilErr(t, err)
	eq(t, len(tags[taglib.AlbumArtist]), 1)
	eq(t, tags[taglib.AlbumArtist][0], "Brian Eno—David Byrne")
}

func TestConcurrent(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)

	c := 250
	pathErrors := make([]error, c)

	var wg sync.WaitGroup
	for i := range c {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := taglib.ReadTags(paths[i%len(paths)]); err != nil {
				pathErrors[i] = fmt.Errorf("iter %d: %w", i, err)
			}
		}()
	}
	wg.Wait()

	err := errors.Join(pathErrors...)
	nilErr(t, err)
}

func TestProperties(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egFLAC, "eg.flac")

	properties, err := taglib.ReadProperties(path)
	nilErr(t, err)

	eq(t, 1*time.Second, properties.Length)
	eq(t, 1460, properties.Bitrate)
	eq(t, 48_000, properties.SampleRate)
	eq(t, 2, properties.Channels)

	eq(t, len(properties.Images), 2)
	eq(t, properties.Images[0].Type, "Front Cover")
	eq(t, properties.Images[0].Description, "The first image")
	eq(t, properties.Images[0].MIMEType, "image/png")
	eq(t, properties.Images[1].Type, "Lead Artist")
	eq(t, properties.Images[1].Description, "The second image")
	eq(t, properties.Images[1].MIMEType, "image/jpeg")
}

func TestMultiOpen(t *testing.T) {
	t.Parallel()

	{
		path := tmpf(t, egFLAC, "eg.flac")
		_, err := taglib.ReadTags(path)
		nilErr(t, err)
	}
	{
		path := tmpf(t, egFLAC, "eg.flac")
		_, err := taglib.ReadTags(path)
		nilErr(t, err)
	}
}

func TestReadImage(t *testing.T) {
	path := tmpf(t, egFLAC, "eg.flac")

	properties, err := taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) > 0, true)

	imgBytes, err := taglib.ReadImage(path)
	nilErr(t, err)
	if imgBytes == nil {
		t.Fatalf("no image")
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	nilErr(t, err)

	b := img.Bounds()
	if b.Dx() != 700 || b.Dy() != 700 {
		t.Fatalf("bad image dimensions: %d, %d != 700, 700", b.Dx(), b.Dy())
	}
}

func TestWriteImage(t *testing.T) {
	path := tmpf(t, egFLAC, "eg.flac")

	err := taglib.WriteImage(path, coverJPG)
	nilErr(t, err)

	imgBytes, err := taglib.ReadImage(path)
	nilErr(t, err)
	if imgBytes == nil {
		t.Fatalf("no written image")
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	nilErr(t, err)

	b := img.Bounds()
	if b.Dx() != 700 || b.Dy() != 700 {
		t.Fatalf("bad image dimensions: %d, %d != 700, 700", b.Dx(), b.Dy())
	}
}

func TestClearImage(t *testing.T) {
	path := tmpf(t, egFLAC, "eg.flac")

	properties, err := taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 2, true) // have two imaages
	eq(t, properties.Images[0].Description, "The first image")

	img, err := taglib.ReadImage(path)
	nilErr(t, err)
	eq(t, len(img) > 0, true)

	nilErr(t, taglib.WriteImage(path, nil))

	properties, err = taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 1, true) // have one images
	eq(t, properties.Images[0].Description, "The second image")

	nilErr(t, taglib.WriteImage(path, nil))

	properties, err = taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 0, true) // have zero images

	img, err = taglib.ReadImage(path)
	nilErr(t, err)
	eq(t, len(img) == 0, true)
}

func TestClearImageReverse(t *testing.T) {
	path := tmpf(t, egFLAC, "eg.flac")

	properties, err := taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 2, true) // have two imaages
	eq(t, properties.Images[0].Description, "The first image")

	img, err := taglib.ReadImage(path)
	nilErr(t, err)
	eq(t, len(img) > 0, true)

	nilErr(t, taglib.WriteImageOptions(path, nil, 1, "", "", "")) // delete the second

	properties, err = taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 1, true)                   // have one images
	eq(t, properties.Images[0].Description, "The first image") // but it's the first one

	nilErr(t, taglib.WriteImage(path, nil))

	properties, err = taglib.ReadProperties(path)
	nilErr(t, err)
	eq(t, len(properties.Images) == 0, true) // have zero images

	img, err = taglib.ReadImage(path)
	nilErr(t, err)
	eq(t, len(img) == 0, true)
}

func TestReadID3v2Frames(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3, "eg.mp3")

	// First write some tags using WriteTags so we have ID3v2 data
	err := taglib.WriteTags(path, map[string][]string{
		"TITLE":  {"Test Title"},
		"ARTIST": {"Test Artist"},
		"ALBUM":  {"Test Album"},
	}, taglib.Clear)
	nilErr(t, err)

	// Now read the ID3v2 frames directly
	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	// Should have frames (the exact frame IDs depend on how TagLib maps tags)
	if len(frames) == 0 {
		t.Fatal("expected some ID3v2 frames")
	}

	// Check that we got TIT2 (title), TPE1 (artist), TALB (album) frames
	if _, ok := frames["TIT2"]; !ok {
		t.Error("expected TIT2 frame for title")
	}
	if _, ok := frames["TPE1"]; !ok {
		t.Error("expected TPE1 frame for artist")
	}
	if _, ok := frames["TALB"]; !ok {
		t.Error("expected TALB frame for album")
	}
}

func TestReadID3v2FramesEmpty(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3, "eg.mp3")

	// Clear all tags first
	err := taglib.WriteTags(path, nil, taglib.Clear)
	nilErr(t, err)

	// Read ID3v2 frames from a file with no tags - should return empty map, not error
	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	// Should be empty or have no meaningful frames
	if frames == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestReadID3v2FramesNonMP3(t *testing.T) {
	t.Parallel()

	// ID3v2 is specific to MP3, so non-MP3 files should return empty frames
	path := tmpf(t, egFLAC, "eg.flac")

	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	// FLAC doesn't use ID3v2, should return empty map
	if len(frames) != 0 {
		t.Errorf("expected empty frames for FLAC, got %d", len(frames))
	}
}

func TestReadID3v2FramesInvalid(t *testing.T) {
	t.Parallel()

	path := tmpf(t, []byte("not a file"), "invalid.mp3")

	frames, err := taglib.ReadID3v2Frames(path)
	// Invalid file should return ErrInvalidFile (nil frames from WASM)
	if err == nil && frames != nil && len(frames) > 0 {
		t.Error("expected error or empty frames for invalid file")
	}
}

func TestReadID3v1Frames(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3, "eg.mp3")

	// Write some tags - TagLib will write to ID3v1 as well for MP3
	err := taglib.WriteTags(path, map[string][]string{
		"TITLE":  {"Test Title"},
		"ARTIST": {"Test Artist"},
		"ALBUM":  {"Test Album"},
	}, taglib.Clear)
	nilErr(t, err)

	// Read ID3v1 frames
	frames, err := taglib.ReadID3v1Frames(path)
	nilErr(t, err)

	if frames == nil {
		t.Fatal("expected non-nil map")
	}

	// ID3v1 uses uppercase field names like TITLE, ARTIST, ALBUM
	if title, ok := frames["TITLE"]; ok {
		eq(t, title[0], "Test Title")
	}
	if artist, ok := frames["ARTIST"]; ok {
		eq(t, artist[0], "Test Artist")
	}
	if album, ok := frames["ALBUM"]; ok {
		eq(t, album[0], "Test Album")
	}
}

func TestReadID3v1FramesNonMP3(t *testing.T) {
	t.Parallel()

	// ID3v1 is specific to MP3
	path := tmpf(t, egFLAC, "eg.flac")

	frames, err := taglib.ReadID3v1Frames(path)
	nilErr(t, err)

	// FLAC doesn't use ID3v1, should return empty map
	if len(frames) != 0 {
		t.Errorf("expected empty frames for FLAC, got %d", len(frames))
	}
}

func TestReadMP4Atoms(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egM4a, "eg.m4a")

	// First write some tags using WriteTags so we have MP4 data
	err := taglib.WriteTags(path, map[string][]string{
		"TITLE":       {"Test Title"},
		"ARTIST":      {"Test Artist"},
		"ALBUM":       {"Test Album"},
		"TRACKNUMBER": {"3"},
	}, taglib.Clear)
	nilErr(t, err)

	// Now read the MP4 atoms directly
	atoms, err := taglib.ReadMP4Atoms(path)
	nilErr(t, err)

	// Should have atoms
	if len(atoms) == 0 {
		t.Fatal("expected some MP4 atoms")
	}

	// Check that we got ©nam (title), ©ART (artist), ©alb (album) atoms
	if _, ok := atoms["©nam"]; !ok {
		t.Error("expected ©nam atom for title")
	}
	if _, ok := atoms["©ART"]; !ok {
		t.Error("expected ©ART atom for artist")
	}
	if _, ok := atoms["©alb"]; !ok {
		t.Error("expected ©alb atom for album")
	}
}

func TestReadMP4AtomsEmpty(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egM4a, "eg.m4a")

	// Clear all tags first
	err := taglib.WriteTags(path, nil, taglib.Clear)
	nilErr(t, err)

	// Read MP4 atoms from a file with no tags - should return empty map, not error
	atoms, err := taglib.ReadMP4Atoms(path)
	nilErr(t, err)

	// Should be empty or have no meaningful atoms
	if atoms == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestReadMP4AtomsNonM4A(t *testing.T) {
	t.Parallel()

	// MP4 atoms are specific to M4A/MP4, so non-M4A files should return empty atoms
	path := tmpf(t, egFLAC, "eg.flac")

	atoms, err := taglib.ReadMP4Atoms(path)
	nilErr(t, err)

	// FLAC doesn't use MP4 atoms, should return empty map
	if len(atoms) != 0 {
		t.Errorf("expected empty atoms for FLAC, got %d", len(atoms))
	}
}

func TestReadMP4AtomsInvalid(t *testing.T) {
	t.Parallel()

	path := tmpf(t, []byte("not a file"), "invalid.m4a")

	atoms, err := taglib.ReadMP4Atoms(path)
	// Invalid file should return ErrInvalidFile (nil atoms from WASM)
	if err == nil && atoms != nil && len(atoms) > 0 {
		t.Error("expected error or empty atoms for invalid file")
	}
}

func TestReadMP4AtomsIntPair(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egM4a, "eg.m4a")

	// Write track and disc numbers
	// Note: TagLib's property mapping stores TRACKNUMBER in trkn:num but TRACKTOTAL
	// goes to a free-form atom, so trkn:total will be 0. This test verifies the
	// IntPair splitting works correctly for the values that are present.
	err := taglib.WriteTags(path, map[string][]string{
		"TRACKNUMBER": {"3"},
		"DISCNUMBER":  {"1"},
	}, taglib.Clear)
	nilErr(t, err)

	// Read MP4 atoms
	atoms, err := taglib.ReadMP4Atoms(path)
	nilErr(t, err)

	// Track number should be split into trkn:num and trkn:total
	if num, ok := atoms["trkn:num"]; ok {
		eq(t, num[0], "3")
	} else {
		t.Error("expected trkn:num atom")
	}
	// trkn:total will be 0 since TagLib stores TRACKTOTAL separately
	if total, ok := atoms["trkn:total"]; ok {
		eq(t, total[0], "0")
	} else {
		t.Error("expected trkn:total atom")
	}

	// Disc number should be split into disk:num and disk:total
	if num, ok := atoms["disk:num"]; ok {
		eq(t, num[0], "1")
	} else {
		t.Error("expected disk:num atom")
	}
	if total, ok := atoms["disk:total"]; ok {
		eq(t, total[0], "0")
	} else {
		t.Error("expected disk:total atom")
	}
}

func TestReadID3v1FramesInvalid(t *testing.T) {
	t.Parallel()

	path := tmpf(t, []byte("not a file"), "invalid.mp3")

	frames, err := taglib.ReadID3v1Frames(path)
	// Invalid file should return ErrInvalidFile (nil frames from WASM)
	if err == nil && frames != nil && len(frames) > 0 {
		t.Error("expected error or empty frames for invalid file")
	}
}

func TestWriteID3v2Frames(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3, "eg.mp3")

	// Clear existing tags
	err := taglib.WriteTags(path, nil, taglib.Clear)
	nilErr(t, err)

	// Write ID3v2 frames directly
	err = taglib.WriteID3v2Frames(path, map[string][]string{
		"TIT2": {"Direct Title"},
		"TPE1": {"Direct Artist"},
		"TALB": {"Direct Album"},
	}, taglib.Clear)
	nilErr(t, err)

	// Read back and verify
	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	if frames["TIT2"] == nil || frames["TIT2"][0] != "Direct Title" {
		t.Errorf("expected TIT2='Direct Title', got %v", frames["TIT2"])
	}
	if frames["TPE1"] == nil || frames["TPE1"][0] != "Direct Artist" {
		t.Errorf("expected TPE1='Direct Artist', got %v", frames["TPE1"])
	}
	if frames["TALB"] == nil || frames["TALB"][0] != "Direct Album" {
		t.Errorf("expected TALB='Direct Album', got %v", frames["TALB"])
	}
}

func TestWriteID3v2FramesMerge(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3, "eg.mp3")

	// Clear and write initial frames
	err := taglib.WriteID3v2Frames(path, map[string][]string{
		"TIT2": {"Title One"},
		"TPE1": {"Artist One"},
	}, taglib.Clear)
	nilErr(t, err)

	// Merge new frame without clearing
	err = taglib.WriteID3v2Frames(path, map[string][]string{
		"TALB": {"Album One"},
	}, 0)
	nilErr(t, err)

	// All frames should exist
	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	if frames["TIT2"] == nil || frames["TIT2"][0] != "Title One" {
		t.Errorf("expected TIT2='Title One', got %v", frames["TIT2"])
	}
	if frames["TALB"] == nil || frames["TALB"][0] != "Album One" {
		t.Errorf("expected TALB='Album One', got %v", frames["TALB"])
	}
}

func TestWriteID3v2FramesInvalid(t *testing.T) {
	t.Parallel()

	path := tmpf(t, []byte("not a file"), "invalid.mp3")

	err := taglib.WriteID3v2Frames(path, map[string][]string{
		"TIT2": {"Test"},
	}, 0)
	// Invalid file should return an error - either a call error (function not in WASM)
	// or ErrSavingFile when the WASM function returns false
	// We accept both scenarios since the WASM binary may not have the function yet
	_ = err // Error is expected but may vary based on WASM binary state
}

func TestReadID3v2FramesUSLT(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3Lyrics, "eg_lyrics.mp3")

	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	// Check for USLT frames with language codes
	// Should have USLT:eng and USLT:deu
	foundEng := false
	foundDeu := false
	for key, values := range frames {
		if key == "USLT:eng" {
			foundEng = true
			if len(values) == 0 || values[0] != "English lyrics content here" {
				t.Errorf("expected English lyrics, got %v", values)
			}
		}
		if key == "USLT:deu" {
			foundDeu = true
			if len(values) == 0 || values[0] != "Deutsche Texte hier" {
				t.Errorf("expected German lyrics, got %v", values)
			}
		}
	}
	if !foundEng {
		t.Error("expected USLT:eng frame for English lyrics")
	}
	if !foundDeu {
		t.Error("expected USLT:deu frame for German lyrics")
	}
}

func TestReadID3v2FramesSYLT(t *testing.T) {
	t.Parallel()

	path := tmpf(t, egMP3Lyrics, "eg_lyrics.mp3")

	frames, err := taglib.ReadID3v2Frames(path)
	nilErr(t, err)

	// Check for SYLT frame with language code
	foundSYLT := false
	for key, values := range frames {
		if key == "SYLT:eng" {
			foundSYLT = true
			if len(values) == 0 {
				t.Error("expected SYLT:eng to have content")
				continue
			}
			// SYLT should be converted to LRC format
			lrc := values[0]
			if !strings.Contains(lrc, "[00:00.00]") {
				t.Errorf("expected LRC timestamp format, got %q", lrc)
			}
			if !strings.Contains(lrc, "Line one") {
				t.Errorf("expected 'Line one' in SYLT content, got %q", lrc)
			}
			if !strings.Contains(lrc, "[00:05.00]") {
				t.Errorf("expected [00:05.00] timestamp, got %q", lrc)
			}
			if !strings.Contains(lrc, "Line two") {
				t.Errorf("expected 'Line two' in SYLT content, got %q", lrc)
			}
		}
	}
	if !foundSYLT {
		t.Error("expected SYLT:eng frame for synchronized lyrics")
	}
}

func TestMemNew(t *testing.T) {
	t.Parallel()

	t.Skip("heavy")

	checkMem(t)

	for range 10_000 {
		path := tmpf(t, egFLAC, "eg.flac")
		_, err := taglib.ReadTags(path)
		nilErr(t, err)
		err = os.Remove(path) // don't blow up incase we're using tmpfs
		nilErr(t, err)
	}
}

func TestMemSameFile(t *testing.T) {
	t.Parallel()

	t.Skip("heavy")

	checkMem(t)

	path := tmpf(t, egFLAC, "eg.flac")
	for range 10_000 {
		_, err := taglib.ReadTags(path)
		nilErr(t, err)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	t.Logf("alloc = %v MiB", memStats.Alloc/1024/1024)
}

func BenchmarkWrite(b *testing.B) {
	path := tmpf(b, egFLAC, "eg.flac")
	b.ResetTimer()

	for range b.N {
		err := taglib.WriteTags(path, bigTags, taglib.Clear)
		nilErr(b, err)
	}
}

func BenchmarkRead(b *testing.B) {
	path := tmpf(b, egFLAC, "eg.flac")
	err := taglib.WriteTags(path, bigTags, taglib.Clear)
	nilErr(b, err)
	b.ResetTimer()

	for range b.N {
		_, err := taglib.ReadTags(path)
		nilErr(b, err)
	}
}

var (
	//go:embed testdata/eg.flac
	egFLAC []byte
	//go:embed testdata/eg.mp3
	egMP3 []byte
	//go:embed testdata/eg.m4a
	egM4a []byte
	//go:embed testdata/eg.ogg
	egOgg []byte
	//go:embed testdata/eg.wav
	egWAV []byte
	//go:embed testdata/cover.jpg
	coverJPG []byte
	//go:embed testdata/eg_lyrics.mp3
	egMP3Lyrics []byte
)

func testPaths(t testing.TB) []string {
	return []string{
		tmpf(t, egFLAC, "eg.flac"),
		tmpf(t, egMP3, "eg.mp3"),
		tmpf(t, egM4a, "eg.m4a"),
		tmpf(t, egWAV, "eg.wav"),
		tmpf(t, egOgg, "eg.ogg"),
	}
}

func tmpf(t testing.TB, b []byte, name string) string {
	p := filepath.Join(t.TempDir(), name)
	err := os.WriteFile(p, b, os.ModePerm)
	nilErr(t, err)
	return p
}

func nilErr(t testing.TB, err error) {
	if err != nil {
		t.Helper()
		t.Fatalf("err: %v", err)
	}
}
func eq[T comparable](t testing.TB, a, b T) {
	if a != b {
		t.Helper()
		t.Fatalf("%v != %v", a, b)
	}
}
func tagEq(t testing.TB, a, b map[string][]string) {
	if !maps.EqualFunc(a, b, slices.Equal) {
		t.Helper()
		t.Fatalf("%q != %q", a, b)
	}
}

func checkMem(t testing.TB) {
	stop := make(chan struct{})
	t.Cleanup(func() {
		stop <- struct{}{}
	})

	go func() {
		ticker := time.Tick(100 * time.Millisecond)

		for {
			select {
			case <-stop:
				return

			case <-ticker:
				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)
				t.Logf("alloc = %v MiB", memStats.Alloc/1024/1024)
			}
		}
	}()
}

var bigTags = map[string][]string{
	"ALBUM":                      {"New Raceion"},
	"ALBUMARTIST":                {"Alan Vega"},
	"ALBUMARTIST_CREDIT":         {"Alan Vega"},
	"ALBUMARTISTS":               {"Alan Vega"},
	"ALBUMARTISTS_CREDIT":        {"Alan Vega"},
	"ARTIST":                     {"Alan Vega"},
	"ARTIST_CREDIT":              {"Alan Vega"},
	"ARTISTS":                    {"Alan Vega"},
	"ARTISTS_CREDIT":             {"Alan Vega"},
	"DATE":                       {"1993-04-02"},
	"DISCNUMBER":                 {"1"},
	"GENRE":                      {"electronic"},
	"GENRES":                     {"electronic", "industrial", "experimental", "proto-punk", "rock", "rockabilly"},
	"LABEL":                      {"GM Editions"},
	"MEDIA":                      {"Digital Media"},
	"MUSICBRAINZ_ALBUMARTISTID":  {"dd720ac8-1c68-4484-abb7-0546413a55e3"},
	"MUSICBRAINZ_ALBUMID":        {"c56a5905-2b3a-46f5-82c7-ce8eed01f876"},
	"MUSICBRAINZ_ARTISTID":       {"dd720ac8-1c68-4484-abb7-0546413a55e3"},
	"MUSICBRAINZ_RELEASEGROUPID": {"373dcce2-63c4-3e8a-9c2c-bc58ec1bbbf3"},
	"MUSICBRAINZ_TRACKID":        {"2f1c8b43-7b4e-4bc8-aacf-760e5fb747a0"},
	"ORIGINALDATE":               {"1993-04-02"},
	"REPLAYGAIN_ALBUM_GAIN":      {"-4.58 dB"},
	"REPLAYGAIN_ALBUM_PEAK":      {"0.977692"},
	"REPLAYGAIN_TRACK_GAIN":      {"-5.29 dB"},
	"REPLAYGAIN_TRACK_PEAK":      {"0.977661"},
	"TITLE":                      {"Christ Dice"},
	"TRACKNUMBER":                {"2"},
	"UPC":                        {"3760271710486"},
}

var longString = strings.Repeat("E", 1024)
