package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"

	"github.com/klauspost/crc32"
)

var defaultIV = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2, 3, 4, 5, 6, 7}

func Crc32(src []byte) uint32 {
	return crc32.ChecksumIEEE(src)
}

func EncodeAes(dst, src, key, iv []byte) {
	if dst == nil {
		dst = src
	}
	if iv == nil {
		iv = defaultIV
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(dst, src)
}

func DecodeAes(dst, src, key, iv []byte) {
	if dst == nil {
		dst = src
	}
	if iv == nil {
		iv = defaultIV
	}
	block, _ := aes.NewCipher(key)
	cipher.NewCFBDecrypter(block, iv).XORKeyStream(dst, src)
}

func EncodeMD5(data []byte) []byte {
	sum := md5.Sum(data)
	return sum[:]

}
