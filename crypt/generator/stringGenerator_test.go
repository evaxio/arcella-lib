package generator

import (
	"fmt"
	"strings"
	"testing"
)

func TestAverage(t *testing.T) {
	fmt.Println("-----------------------------")
	fmt.Println("Hash:		", RandomBase16String(13))
	fmt.Println("Hash:		", RandomBase16String(13))
	fmt.Println("Hash:		", RandomBase16String(13))
	fmt.Println("Hash:		", RandomBase16String(13))
	fmt.Println("Hash:		", RandomBase16String(13))
	fmt.Println("-----------------------------")
	fmt.Println("Hash:		", ShortID(13))
	fmt.Println("Hash:		", ShortID(13))
	fmt.Println("Hash:		", ShortID(13))
	fmt.Println("Hash:		", ShortID(13))
	fmt.Println("Hash:		", ShortID(13))

}

func TestRandomBase16StringLengthAndCharset(t *testing.T) {
	for _, l := range []int{1, 2, 13, 16, 33, 64} {
		s := RandomBase16String(l)
		if len(s) != l {
			t.Errorf("RandomBase16String(%d) length = %d, want %d", l, len(s), l)
		}
		for i, r := range s {
			if !strings.ContainsRune("0123456789abcdef", r) {
				t.Errorf("RandomBase16String(%d)[%d] = %q, not a lowercase hex char", l, i, r)
			}
		}
	}
}

func TestShortIDLengthAndCharset(t *testing.T) {
	for _, l := range []int{1, 7, 13, 64} {
		s := ShortID(l)
		if len(s) != l {
			t.Errorf("ShortID(%d) length = %d, want %d", l, len(s), l)
		}
		for i, r := range s {
			if !strings.ContainsRune(chars, r) {
				t.Errorf("ShortID(%d)[%d] = %q, not in the allowed charset", l, i, r)
			}
		}
	}
}
