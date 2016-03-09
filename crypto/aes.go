package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"hash/crc32"
)

func Crc32(src []byte) uint32 {
	return crc32.ChecksumIEEE(src)
}

func EncodeAes(dst, src, key, iv []byte) {
	if dst == nil {
		dst = src
	}
	block, _ := aes.NewCipher(key)
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(dst, src)
}

func DecodeAes(dst, src, key, iv []byte) {
	if dst == nil {
		dst = src
	}
	block, _ := aes.NewCipher(key)
	cipher.NewCFBDecrypter(block, iv).XORKeyStream(dst, src)
}

func EncodeMD5(data []byte) []byte {
	sum := md5.Sum(data)
	return sum[:]

}
