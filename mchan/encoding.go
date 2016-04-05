package mchan

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/chzyer/next/crypto"
)

type ReplyInfo struct {
	Token   []byte          `json:"token"`
	Code    int             `json:"code,omitempty"`
	Path    string          `json:"path,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type cryptoReply struct {
	Data []byte `json:"data"`
}

type replyError struct {
	Error string `json:"error"`
}

func Send(key []byte, path string, obj interface{}) []byte {
	sent := &ReplyInfo{
		Path: path,
	}
	if obj != nil {
		ret, err := json.Marshal(obj)
		if err != nil {
			panic(err)
		}
		sent.Payload = ret
	}
	return Encode(key, sent)
}

func DecodeReply(key, data []byte, obj interface{}) error {
	reply, err := Decode(key, data)
	if err != nil {
		return err
	}
	return json.Unmarshal(reply.Payload, obj)
}

func Reply(key []byte, obj interface{}) []byte {
	payload, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return Encode(key, &ReplyInfo{
		Code:    200,
		Payload: payload,
	})
}

func ReplyError(key []byte, err error) []byte {
	s := replyError{err.Error()}
	ret, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	return Encode(key, &ReplyInfo{
		Code:    400,
		Payload: ret,
	})
}

func Encode(key []byte, r *ReplyInfo) []byte {
	if r.Payload == nil {
		raw, _ := json.Marshal(nil)
		r.Payload = raw
	}
	if r.Token == nil {
		length := 32
		size := 128
		if len(r.Payload) < size {
			length = size - len(r.Payload)
		}
		r.Token = make([]byte, length)
		rand.Read(r.Token)
	}
	ret, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	crypto.EncodeAes(ret, ret, key, nil)
	ret, _ = json.Marshal(cryptoReply{ret})
	return ret
}

func Decode(key, data []byte) (*ReplyInfo, error) {
	var creply cryptoReply
	err := json.Unmarshal(data, &creply)
	if err != nil {
		return nil, err
	}
	// {"data": "xxxxx=="}

	crypto.DecodeAes(creply.Data, creply.Data, key, nil)
	// {"code": xx, "path": "", "payload": ""}

	var reply ReplyInfo
	err = json.Unmarshal(creply.Data, &reply)
	if err != nil {
		return nil, err
	}

	if reply.Code != 0 && reply.Code/100 != 2 {
		// {"code": 400, "payload": {"error": "xxx"}}
		var replyErr replyError
		err := json.Unmarshal(reply.Payload, &replyErr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(replyErr.Error)
	}

	return &reply, nil
}
