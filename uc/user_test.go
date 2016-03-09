package uc

import (
	"os"
	"testing"

	"github.com/chzyer/next/test"
)

func TestUsers(t *testing.T) {
	defer test.New(t)

	us := NewUsers()
	u1 := us.Register("hello", "bye")
	u2 := us.LoginByName("hello", "bye1")
	test.Nil(u2)
	u2 = us.LoginByName("hello", "bye")
	test.Equal(u1, u2)

	savePath := "/tmp/users.tmp"
	os.Remove(savePath)
	defer os.Remove(savePath)
	err := us.Save(savePath)
	test.Nil(err)
	us2 := NewUsers()
	us2.Load(savePath)
	u3 := us2.LoginByName("hello", "bye")
	test.Equal(u1.UserInfo, u3.UserInfo)
}
