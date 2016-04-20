package packet

import "gopkg.in/logex.v1"

var (
	ErrPortNotMatch = logex.Define("port %v is not matched")
	ErrUserNotMatch = logex.Define("user %v is not matched")
)
