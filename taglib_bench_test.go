package taglib_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.senan.xyz/taglib"
)

// openTestFile opens a file for benchmarking and returns a cleanup function
func openTestFile(b *testing.B, path string) *os.File {
	b.Helper()
	file, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	return file
}

// BenchmarkOpenFile benchmarks opening a file via path
func BenchmarkOpenFile(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

	// Warm up - ensure runtime is initialized
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
}

// BenchmarkOpenStream benchmarks opening a file via io.ReadSeeker (using os.File)
func BenchmarkOpenStream(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

	// Warm up - ensure runtime is initialized
	file := openTestFile(b, path)
	f, err := taglib.OpenStream(file)
	if err != nil {
		b.Fatal(err)
	}
	f.Close()
	file.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := openTestFile(b, path)
		f, err := taglib.OpenStream(file)
		if err != nil {
			b.Fatal(err)
		}
		f.Close()
		file.Close()
	}
}

// BenchmarkReadTagsFile benchmarks reading tags via path
func BenchmarkReadTagsFile(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

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
}

// BenchmarkReadTagsStream benchmarks reading tags via io.ReadSeeker (using os.File)
func BenchmarkReadTagsStream(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

	// Warm up
	file := openTestFile(b, path)
	f, err := taglib.OpenStream(file)
	if err != nil {
		b.Fatal(err)
	}
	_ = f.Tags()
	f.Close()
	file.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := openTestFile(b, path)
		f, err := taglib.OpenStream(file)
		if err != nil {
			b.Fatal(err)
		}
		_ = f.Tags()
		f.Close()
		file.Close()
	}
}

// BenchmarkReadPropertiesFile benchmarks reading audio properties via path
func BenchmarkReadPropertiesFile(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

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
}

// BenchmarkReadPropertiesStream benchmarks reading audio properties via io.ReadSeeker (using os.File)
func BenchmarkReadPropertiesStream(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

	// Warm up
	file := openTestFile(b, path)
	f, err := taglib.OpenStream(file)
	if err != nil {
		b.Fatal(err)
	}
	_ = f.Properties()
	f.Close()
	file.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := openTestFile(b, path)
		f, err := taglib.OpenStream(file)
		if err != nil {
			b.Fatal(err)
		}
		_ = f.Properties()
		f.Close()
		file.Close()
	}
}

// BenchmarkReadAllFile benchmarks reading all tags and properties via path
func BenchmarkReadAllFile(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

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
}

// BenchmarkReadAllStream benchmarks reading all tags and properties via io.ReadSeeker (using os.File)
func BenchmarkReadAllStream(b *testing.B) {
	path := filepath.Join("testdata", "eg.mp3")

	// Warm up
	file := openTestFile(b, path)
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
		file := openTestFile(b, path)
		f, err := taglib.OpenStream(file)
		if err != nil {
			b.Fatal(err)
		}
		_ = f.Tags()
		_ = f.Properties()
		f.Close()
		file.Close()
	}
}
