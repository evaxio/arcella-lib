package bcrypt

import (
	"fmt"
	"testing"
)

func TestAverage(t *testing.T) {

	password := "admin"               // "secret"
	hash, _ := HashPassword(password) // ignore error for the sake of simplicity

	fmt.Println("Password:	", password)
	fmt.Println("Hash:		", hash)

	match := CheckPasswordHash(password, hash)
	fmt.Println("Match:		", match)

	if !match {
		t.Error("Hash is wrong !")
	}

}

// precomputed bcrypt cost-14 hash of "secret" (avoids a 20+ second hash per run)
const precomputedHash = "$2a$14$OzjNC.YwKcDntyWC2UBPBOhpnSx9naZUvOqkj2T0ITCLGZQlnYQS6"

func TestCheckPasswordHashWrongPassword(t *testing.T) {
	if CheckPasswordHash("not-the-password", precomputedHash) {
		t.Errorf("CheckPasswordHash(wrong) = true, want false")
	}
	if CheckPasswordHash("secret", "not-a-hash") {
		t.Errorf("CheckPasswordHash(invalid hash) = true, want false")
	}
}
