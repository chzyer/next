package util

import "io"

func ReadFull(r io.Reader, n int) ([]byte, error) {
	ret := make([]byte, n)
	_, err := io.ReadFull(r, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
