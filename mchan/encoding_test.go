package mchan

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/chzyer/test"
)

type A struct {
	Info string
}

func TestEncoding(t *testing.T) {
	defer test.New(t)

	key := make([]byte, 32)
	rand.Read(key)

	{ // obj
		a := A{
			Info: "hello",
		}
		info := Reply(key, &a)
		// println(string(Reply(key, &a)))
		// println(string(Reply(key, &a)))

		var a2 A
		err := DecodeReply(key, info, &a2)
		test.Nil(err)
		test.Equal(a2, a)
	}

	{ // error
		errFuck := fmt.Errorf("fuck")
		info := ReplyError(key, errFuck)

		err := DecodeReply(key, info, nil)
		test.Equal(err.Error(), errFuck.Error())
	}
}
