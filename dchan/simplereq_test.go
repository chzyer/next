package dchan

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/chzyer/next/util"
	"github.com/chzyer/test"
)

func TestSimpleReq(t *testing.T) {
	defer test.New(t)

	buf := util.RandStr(24)

	req, err := http.NewRequest("GET", "/path", bytes.NewReader([]byte(buf)))
	test.Nil(err)
	req.Header.Set("hello", "bye")

	out := bytes.NewBuffer(nil)
	req.Write(out)

	req2, err := NewSimpleReq(bufio.NewReader(out))
	test.Nil(err)
	test.Equal(req.Header.Get("hello"), req2.Header.Get("hello"))
	body, err := ioutil.ReadAll(req2.Body)
	test.Nil(err)
	test.Equal(body, []byte(buf))
}
