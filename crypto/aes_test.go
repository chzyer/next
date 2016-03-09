package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/chzyer/next/test"
)

func TestDeEnCode(t *testing.T) {
	defer test.New(t)

	src := make([]byte, 24)
	rand.Read(src)
	key := make([]byte, 32)
	iv := make([]byte, 16)
	rand.Read(key)
	rand.Read(iv)
	dst := make([]byte, 24)
	EncodeAes(dst, src, key, iv)
	DecodeAes(dst, dst, key, iv)
	test.Equal(src, dst)
}
