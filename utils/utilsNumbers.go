package utils

import (
	"encoding/binary"
	"math"
	"strconv"
)

// Deprecated: no in-repo consumers; kept for compatibility.
func SieveOfEratosthenes(M, N int) (primes []int) {
	if N < 3 {
		return nil
	}
	composite := make([]bool, N)
	for i := 2; i*i < N; i++ {
		if !composite[i] {
			for j := i * i; j < N; j += i {
				composite[j] = true
			}
		}
	}
	if M < 2 {
		M = 2
	}
	for j := M; j < N; j++ {
		if !composite[j] {
			primes = append(primes, j)
		}
	}
	return
}

func Float64ToByte(f float64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
	return buf[:]
}

func TruncateFloat(f float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))
	if precision < 0 {
		multiplier = 1
	}
	return math.Trunc(f*multiplier) / multiplier
}

func HexToInt64Default(ident string, i int64) int64 {
	if value, err := strconv.ParseInt(ident, 16, 64); err != nil {
		return i
	} else {
		return value
	}
}
