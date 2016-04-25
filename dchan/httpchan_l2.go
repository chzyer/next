package dchan

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

var (
	HttpKeyIV       = "X-Log"
	HttpKeyUserId   = "X-Host-ID"
	HttpKeyChecksum = "X-MD5"
)

func (h *HttpChan) ReadL2(b *bufio.Reader) (*packet.PacketL2, error) {
	req, err := http.ReadRequest(b)
	if err != nil {
		return nil, logex.Trace(err, "read l2 request")
	}
	iv, err := base64.URLEncoding.DecodeString(req.Header.Get(HttpKeyIV))
	if err != nil {
		return nil, logex.Trace(err, "error in decode iv")
	}
	userId, err := strconv.Atoi(req.Header.Get(HttpKeyUserId))
	if err != nil {
		return nil, logex.Trace(err, "error in decode userid")
	}
	checksum, err := strconv.Atoi(req.Header.Get(HttpKeyChecksum))

	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, logex.Trace(err, "read l2 payload")
	}

	return packet.NewPacketL2(iv, uint16(userId), payload, uint32(checksum)), nil
}

func (h *HttpChan) MarshalL2(p *packet.PacketL2) []byte {
	body := bytes.NewReader(p.Payload)
	req, err := http.NewRequest("POST", "/data", body)
	if err != nil {
		panic(err)
	}

	req.Header.Set(HttpKeyIV, base64.URLEncoding.EncodeToString(p.IV))
	req.Header.Set(HttpKeyUserId, strconv.Itoa(int(p.UserId)))
	req.Header.Set(HttpKeyChecksum, strconv.Itoa(int(p.Checksum)))
	req.Header.Set("Content-Type", "application/octet-stream")
	out := bytes.NewBuffer(nil)
	req.Write(out)
	return out.Bytes()
}
