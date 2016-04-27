package ip

import (
	"encoding/binary"
	"net"
	"reflect"
	"strings"

	"github.com/chzyer/flagly"
	"github.com/chzyer/logex"
)

type IP [4]byte

func IsIP(s string) bool {
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	rs := []rune(s)
	for _, r := range rs {
		if (r >= '0' && r <= '9') || r == '.' {
			continue
		}
		return false
	}
	return true
}

func NewIP(b []byte) (ret IP) {
	copy(ret[:], b)
	return
}

func (i IP) Equal(i2 IP) bool {
	return i[3] == i2[3] &&
		i[2] == i2[2] &&
		i[1] == i2[1] &&
		i[0] == i2[0]
}

func (i IP) String() string {
	return i.IP().String()
}

func MatchIPNet(child, parent *net.IPNet) bool {
	childOne, _ := child.Mask.Size()
	parentOne, _ := parent.Mask.Size()
	if childOne < parentOne {
		// child has a bigger subnet
		return false
	}
	return parent.Contains(child.IP)
}

type IPNet struct {
	IP   IP         // network number
	Mask net.IPMask // network mask
}

func (i *IPNet) ToNet() *net.IPNet {
	return &net.IPNet{
		IP:   i.IP.IP(),
		Mask: i.Mask,
	}
}

func (IPNet) ParseArgs(args []string) (reflect.Value, error) {
	n, err := ParseCIDR(args[0])
	if err != nil {
		return flagly.NilValue, err
	}
	return reflect.ValueOf(n), nil
}
func (IPNet) Type() reflect.Type { return reflect.TypeOf(IPNet{}) }

func (in *IPNet) String() string {
	ipn := &net.IPNet{
		IP:   in.IP.IP(),
		Mask: in.Mask,
	}
	return ipn.String()
}

func ParseCIDR(s string) (*IPNet, error) {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, logex.Trace(err)
	}
	return &IPNet{
		IP:   CopyIP(ip.To4()),
		Mask: ipnet.Mask,
	}, nil
}

func ParseIntIP(ip uint32) IP {
	var ret IP
	binary.BigEndian.PutUint32(ret[:], ip)
	return ret
}

func (ip IP) Clone() IP {
	return ip
}

func (ip IP) Int() uint32 {
	return binary.BigEndian.Uint32(ip[:])
}

func (ip IP) IP() net.IP {
	return net.IPv4(ip[0], ip[1], ip[2], ip[3])
}

func ParseIP(s string) IP {
	return CopyIP(net.ParseIP(s))
}

func CopyIP(ip net.IP) IP {
	ret := [4]byte{}
	copy(ret[:], ip.To4())
	return IP(ret)
}
