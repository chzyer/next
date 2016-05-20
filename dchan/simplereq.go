package dchan

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

type SimpleReq struct {
	Header Header
	Body   io.ReadCloser
}

type Header []header

func (h Header) get(k string) int {
	for idx := range h {
		if strings.EqualFold(h[idx].Name, k) {
			return idx
		}
	}
	return -1
}

func (h Header) Get(k string) string {
	idx := h.get(k)
	if idx < 0 {
		return ""
	}
	return h[idx].Values[0]
}

func (h *Header) Add(k string, v string) {
	idx := h.get(k)
	if idx < 0 {
		*h = append(*h, header{k, []string{v}})
	} else {
		(*h)[idx].Values = append((*h)[idx].Values, v)
	}
}

type header struct {
	Name   string
	Values []string
}

func NewSimpleReq(r *bufio.Reader) (*SimpleReq, error) {
	sr := &SimpleReq{
		Header: make(Header, 0, 16),
	}

	for {
		line, err := r.ReadSlice('\n')
		if err != nil {
			return nil, err
		}
		if len(line) == 2 {
			break
		}

		if idx := bytes.Index(line, []byte(":")); idx > 0 {
			sr.Header.Add(string(line[:idx]), string(bytes.TrimSpace(line[idx+1:])))
		}
	}

	cl := sr.Header.Get("Content-Length")
	if n, _ := strconv.Atoi(cl); n > 0 {
		sr.Body = ioutil.NopCloser(io.LimitReader(r, int64(n)))
	} else {
		sr.Body = ioutil.NopCloser(r)
	}
	return sr, nil
}
