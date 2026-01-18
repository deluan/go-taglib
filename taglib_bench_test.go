package taglib_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.senan.xyz/taglib"
)

var testFiles = []string{
	"eg.mp3",
	"eg.flac",
	"eg.m4a",
	"eg.ogg",
	"eg.wav",
	"eg.aiff",
}

func BenchmarkOpen(b *testing.B) {
	for _, name := range testFiles {
		path := filepath.Join("testdata", name)

		b.Run("File/"+name, func(b *testing.B) {
			// Warm up
			f, err := taglib.OpenReadOnly(path)
			if err != nil {
				b.Fatal(err)
			}
			f.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f, err := taglib.OpenReadOnly(path)
				if err != nil {
					b.Fatal(err)
				}
				f.Close()
			}
		})

		b.Run("Stream/"+name, func(b *testing.B) {
			// Warm up
			file, _ := os.Open(path)
			f, err := taglib.OpenStream(file)
			if err != nil {
				b.Fatal(err)
			}
			f.Close()
			file.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file, _ := os.Open(path)
				f, err := taglib.OpenStream(file)
				if err != nil {
					b.Fatal(err)
				}
				f.Close()
				file.Close()
			}
		})
	}
}

func BenchmarkReadTags(b *testing.B) {
	for _, name := range testFiles {
		path := filepath.Join("testdata", name)

		b.Run("File/"+name, func(b *testing.B) {
			// Warm up
			f, err := taglib.OpenReadOnly(path)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Tags()
			f.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f, err := taglib.OpenReadOnly(path)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Tags()
				f.Close()
			}
		})

		b.Run("Stream/"+name, func(b *testing.B) {
			// Warm up
			file, _ := os.Open(path)
			f, err := taglib.OpenStream(file)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Tags()
			f.Close()
			file.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file, _ := os.Open(path)
				f, err := taglib.OpenStream(file)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Tags()
				f.Close()
				file.Close()
			}
		})
	}
}

func BenchmarkReadProperties(b *testing.B) {
	for _, name := range testFiles {
		path := filepath.Join("testdata", name)

		b.Run("File/"+name, func(b *testing.B) {
			// Warm up
			f, err := taglib.OpenReadOnly(path)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Properties()
			f.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f, err := taglib.OpenReadOnly(path)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Properties()
				f.Close()
			}
		})

		b.Run("Stream/"+name, func(b *testing.B) {
			// Warm up
			file, _ := os.Open(path)
			f, err := taglib.OpenStream(file)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Properties()
			f.Close()
			file.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file, _ := os.Open(path)
				f, err := taglib.OpenStream(file)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Properties()
				f.Close()
				file.Close()
			}
		})
	}
}

func BenchmarkReadAll(b *testing.B) {
	for _, name := range testFiles {
		path := filepath.Join("testdata", name)

		b.Run("File/"+name, func(b *testing.B) {
			// Warm up
			f, err := taglib.OpenReadOnly(path)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Tags()
			_ = f.Properties()
			f.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f, err := taglib.OpenReadOnly(path)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Tags()
				_ = f.Properties()
				f.Close()
			}
		})

		b.Run("Stream/"+name, func(b *testing.B) {
			// Warm up
			file, _ := os.Open(path)
			f, err := taglib.OpenStream(file)
			if err != nil {
				b.Fatal(err)
			}
			_ = f.Tags()
			_ = f.Properties()
			f.Close()
			file.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file, _ := os.Open(path)
				f, err := taglib.OpenStream(file)
				if err != nil {
					b.Fatal(err)
				}
				_ = f.Tags()
				_ = f.Properties()
				f.Close()
				file.Close()
			}
		})
	}
}
