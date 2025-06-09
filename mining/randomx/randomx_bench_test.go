//go:build cgo
// +build cgo

package randomx

import (
	"testing"
)

func BenchmarkRandomXLight(b *testing.B) {
	cache, err := NewCache([]byte("benchmark seed"))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	vm, err := NewVM(cache, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer vm.Close()

	input := []byte("benchmark input")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vm.CalcHash(input)
	}
}

func BenchmarkRandomXFull(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping full dataset benchmark in short mode")
	}

	cache, err := NewCache([]byte("benchmark seed"))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	dataset, err := NewDataset(cache)
	if err != nil {
		b.Fatal(err)
	}
	defer dataset.Close()

	vm, err := NewVM(cache, dataset)
	if err != nil {
		b.Fatal(err)
	}
	defer vm.Close()

	input := []byte("benchmark input")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vm.CalcHash(input)
	}
}
