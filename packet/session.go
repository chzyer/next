package packet

import (
	"github.com/chzyer/next/crypto"
	"github.com/chzyer/next2/util"
)

type AuthDelegate interface {
	GetUserToken(userId int) ([]byte, error)
}

type Session struct {
	delegate AuthDelegate

	userId int
	token  []byte
}

func NewSessionSvr(delegate AuthDelegate) *Session {
	return &Session{
		delegate: delegate,
		userId:   -1,
	}
}

func NewSessionCli(userId int, token []byte) *Session {
	return &Session{
		userId: userId,
		token:  token,
	}
}

func (s *Session) Clone() *Session {
	return &Session{
		delegate: s.delegate,
		userId:   s.userId,
		token:    s.token,
	}
}

func (s *Session) Verify(userId int, crc32 uint32, iv, payload []byte) error {
	if err := s.VerifyUserId(userId); err != nil {
		return err
	}
	s.Decode(iv, payload, payload)
	if util.Crc32(payload) != crc32 {
		return ErrInvalidToken.Trace("checksum not match")
	}
	return nil
}

func (s *Session) UserId() int {
	if s.userId < 0 {
		panic("session is not inited")
	}
	return s.userId
}

// svr method
func (s *Session) VerifyUserId(userId int) error {
	if s.userId >= 0 {
		if userId != s.userId {
			return ErrUserNotMatch.Trace()
		}
		return nil
	}

	token, err := s.delegate.GetUserToken(userId)
	if err != nil {
		return err
	}
	s.userId = userId
	s.token = token
	return nil
}

func (s *Session) Encode(iv, dst, src []byte) {
	if s.token == nil {
		panic("session is not inited, token is nil")
	}
	crypto.EncodeAes(dst, src, s.token, iv)
}

func (s *Session) Decode(iv []byte, dst, src []byte) {
	crypto.DecodeAes(dst, src, s.token, iv)
}
