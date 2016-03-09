package uc

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/chzyer/next/crypto"
	"gopkg.in/logex.v1"
)

var (
	ErrInvalidAuthToken = logex.Define("invalid auth token")
)

type AuthData struct {
	UserName string
	Passcode []byte
}

// Token = aes_256_cfb(md5(pswd) + ":" + timestamp, key, iv)
// key = specified key by config
// iv = base64(rand(16))
type AuthRequest struct {
	UserName string `json:"username"`
	Token    []byte `json:"token"`
	IV       []byte `json:"iv"`
}

// passcode: sha1(password + salt)
func NewAuthRequest(userName string, timestamp int64, passcode, key []byte) *AuthRequest {
	iv := make([]byte, 16)
	rand.Read(iv)

	// token = aes(passcode + timestamp)
	token := make([]byte, len(passcode)+8) // timestamp
	copy(token, passcode)
	binary.BigEndian.PutUint64(token[len(passcode):], uint64(timestamp))
	crypto.EncodeAes(token, token, key, iv)

	return &AuthRequest{
		UserName: userName,
		Token:    token,
		IV:       iv,
	}
}

func (a *AuthRequest) Decode(key []byte, nowTime int64) (*AuthData, error) {
	token := make([]byte, len(a.Token))
	crypto.DecodeAes(token, a.Token, key, a.IV)
	timeSent := int64(binary.BigEndian.Uint64(token[len(token)-8:]))
	if nowTime-timeSent > 20 || timeSent > nowTime { // 20s expired time
		return nil, ErrInvalidAuthToken.Trace()
	}
	return &AuthData{
		UserName: a.UserName,
		Passcode: token[:len(token)-8],
	}, nil
}
