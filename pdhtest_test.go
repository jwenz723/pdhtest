// +build windows

package main

import (
	"testing"
)

func BenchmarkCounterPath(b *testing.B) {
	path := `\Processor(*)\% Processor Time`
	for i := 0; i < 200; i++ {
		_, _ = CounterPath(path)
	}
}
