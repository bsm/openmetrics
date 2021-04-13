package metro_test

import (
	"testing"

	. "github.com/bsm/openmetrics/internal/metro"
)

func TestHashString(t *testing.T) {
	examples := []struct {
		Input        string
		Seed, Expect uint64
	}{
		{"", 0, 8097384203561113213},
		{"", 1, 16642596117950257669},
		{"hello", 0, 4571129541730210044},
		{"hello", 4571129541730210044, 13883141454884313713},
		{"日本国", 0, 4962688877382791512},
	}

	for i, x := range examples {
		if exp, got := x.Expect, HashString(x.Input, x.Seed); exp != got {
			t.Errorf("[%d] for %q [%d] expected %v, got %v", i, x.Input, x.Seed, exp, got)
		}
	}
}

func TestHashByte(t *testing.T) {
	examples := []struct {
		Input        byte
		Seed, Expect uint64
	}{
		{0, 0, 1044577374344929784},
		{'x', 0, 11925460358279924827},
		{'x', 11925460358279924827, 10320758655387219728},
		{255, 1, 5031988715912559013},
	}

	for i, x := range examples {
		if exp, got := x.Expect, HashByte(x.Input, x.Seed); exp != got {
			t.Errorf("[%d] for %q [%d] expected %v, got %v", i, x.Input, x.Seed, exp, got)
		}
	}
}

func BenchmarkHashString(b *testing.B) {
	p := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	for i := 0; i < b.N; i++ {
		HashString(p, 0)
	}
}

func BenchmarkHashByte(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashByte('x', 0)
	}
}
