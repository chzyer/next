package server

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/chzyer/next/uc"
	"github.com/chzyer/test"
)

func TestAuthRequest(t *testing.T) {
	defer test.New(t)

	authData := &uc.AuthData{
		UserName: "hello",
		Passcode: []byte("password"),
	}
	timestamp := time.Now().Unix()
	key := make([]byte, 32)
	rand.Read(key)
	authReq := uc.NewAuthRequest(authData.UserName, timestamp, authData.Passcode, key)
	authData2, err := authReq.Decode(key, timestamp+21)
	test.Nil(authData2)
	test.Equal(err, uc.ErrInvalidAuthToken)
	authData2, err = authReq.Decode(key, timestamp+19)
	test.Nil(err)
	test.Equal(authData2, authData)
}
