package uuid

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"
)

const bytesLen = 16
const nanoPerMilli = int64(time.Millisecond)

// UUID7 https://dev.to/siddhantkcode/identifiers-101-understanding-and-implementing-uuids-and-ulids-2kc6
type UUID7 struct {
	uuid [bytesLen]byte // uuid value
}

func NewUUID7() *UUID7 {
	id := UUID7{}
	id.NewUUID7()
	return &id
}

func (u *UUID7) NewUUID7() {
	timestamp := uint64(time.Now().UnixNano() / nanoPerMilli)
	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], timestamp)
	copy(u.uuid[:6], tsBuf[:6])

	randomBytes := make([]byte, 10)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(err)
	}
	copy(u.uuid[6:], randomBytes)

	// Set version (7) and variant bits (2 MSB as 01)
	u.uuid[6] = (u.uuid[6] & 0x0f) | (7 << 4)
	u.uuid[8] = (u.uuid[8] & 0x3f) | 0x80
}

func (u *UUID7) Base64() string {
	return base64.StdEncoding.EncodeToString(u.uuid[:])
}

func (u *UUID7) Bytes() []byte { return u.uuid[:] }

func (u *UUID7) String() string {
	var dst [36]byte
	hex.Encode(dst[:], u.uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], u.uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], u.uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], u.uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], u.uuid[10:])
	return string(dst[:])
}

func (u *UUID7) xorIt(in [bytesLen]byte, key []byte) []byte {
	eb := make([]byte, bytesLen)
	for i := 0; i < bytesLen; i++ {
		eb[i] = in[i] ^ key[i%bytesLen]
	}
	return eb
}

// XorEncrypt obfuscates a UUID with a random XOR key. The key is appended to
// the output, so the result is reversible by anyone holding it; this is
// obfuscation, not encryption.
func (u *UUID7) XorEncrypt() string {
	key := make([]byte, bytesLen)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	ct := u.xorIt(u.uuid, key)
	out := make([]byte, bytesLen*2)
	copy(out, ct)
	copy(out[bytesLen:], key)
	return base64.StdEncoding.EncodeToString(out)
}

func (u *UUID7) XorDecrypt(data string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	if len(raw) != bytesLen*2 {
		return "", errors.New("invalid xored uuid length")
	}
	var in [bytesLen]byte
	copy(in[:], raw[:bytesLen])
	key := raw[bytesLen:]
	dec := u.xorIt(in, key)
	id := UUID7{uuid: [bytesLen]byte(dec)}
	return id.String(), nil
}
