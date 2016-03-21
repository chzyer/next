package uc

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

var (
	ErrUserNotFound = logex.Define("user not found")
)

type Users struct {
	user []User
	m    sync.Mutex
}

func NewUsers() *Users {
	return &Users{}
}

func (us *Users) Register(name string, pswd string) *User {
	ui := &UserInfo{
		Name:     name,
		Password: pswd,
	}
	return us.AddUser(ui)
}

func (us *Users) LoginByName(name string, pswd string) *User {
	for idx, u := range us.user {
		if u.Name == name {
			return us.Login(idx, pswd)
		}
	}
	return nil
}

func (us *Users) Show() []User {
	return us.user
}

func (us *Users) Find(username string) *User {
	for _, u := range us.user {
		if u.Name == username {
			return &u
		}
	}
	return nil
}

func (us *Users) Login(userId int, pswd string) *User {
	u := us.FindId(userId)
	if u == nil {
		return nil
	}
	if u.Password != pswd {
		return nil
	}
	return u
}

func (us *Users) AddUser(ui *UserInfo) *User {
	us.m.Lock()
	u := NewUser(ui)
	u.Id = uint16(len(us.user))
	us.user = append(us.user, *u)
	us.m.Unlock()
	return u
}

func (u *Users) Load(fp string) error {
	mutex := u.m
	mutex.Lock()
	defer mutex.Unlock()

	fh, err := os.OpenFile(fp, os.O_RDONLY, 0600)
	if err != nil {
		return logex.Trace(err)
	}
	defer fh.Close()

	return logex.Trace(gob.NewDecoder(fh).Decode(&u.user))
}

func (u *Users) Save(fp string) error {
	u.m.Lock()
	defer u.m.Unlock()

	fh, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return logex.Trace(err)
	}
	defer fh.Close()

	return logex.Trace(gob.NewEncoder(fh).Encode(u.user))
}

func (us *Users) FindId(id int) *User {
	if id >= len(us.user) {
		return nil
	}
	u := &us.user[id]
	if u.Id == 0 {
		u.Id = uint16(id)
	}
	return u
}

type User struct {
	*UserInfo
	Net   *ip.IP
	Token string
	in    chan *packet.Packet
	out   chan *packet.Packet
}

func NewUser(ui *UserInfo) *User {
	return &User{
		UserInfo: ui,
		Token:    GenToken(),
	}
}

func (u *User) GetChannel() (in, out chan *packet.Packet) {
	if u.in == nil {
		u.in = make(chan *packet.Packet)
	}
	if u.out == nil {
		u.out = make(chan *packet.Packet)
	}
	return u.in, u.out
}

func (u User) String() string {
	return fmt.Sprintf(`{Id: %v, Name: %v, Token: %v, Net: %v, IsAdmin: %v}`,
		u.Id, u.Name, u.Token, u.Net, u.IsAdmin)
}

// directly encode UserInfo to ignore other temporary variables
func (u *User) GobDecode(data []byte) error {
	var ui *UserInfo
	err := gob.NewDecoder(bytes.NewBuffer(data)).Decode(&ui)
	if err != nil {
		return err
	}
	newUser := NewUser(ui)
	*u = *newUser
	return nil
}

func (u *User) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(u.UserInfo); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *User) IsOnline() bool {
	return u.Net != nil
}

type UserInfo struct {
	Id       uint16
	Name     string
	Password string
	IsAdmin  bool
}

func init() {
	rand.Seed(time.Now().Unix())
}

func GenToken() string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]byte, 30)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
