package packet

import "testing"

func TestIV(t *testing.T) {
	s := NewSessionIV(1, 1024, make([]byte, 30))
	ivb := s.GenIV()
	iv := ParseIV(ivb)
	if s.Port != iv.Port || s.UserId != iv.UserId || iv.ReqId != GetReqId() {
		t.Fatal("error")
	}
}
