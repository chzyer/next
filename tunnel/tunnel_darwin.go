package tunnel

import (
	"errors"
	"fmt"
	"net"
	"os"

	"gopkg.in/logex.v1"
)

// #include "tunnel_darwin.c"
import "C"

func OpenTun(idx int) (*os.File, error) {
	fd := C.utun_open(C.int(idx))
	if fd < 0 {
		return nil, errors.New(C.GoString(C.strerror(-fd)))
	}
	return os.NewFile(uintptr(fd), fmt.Sprintf("/dev/tun%d", idx)), nil
}

func (t *Instance) setupTun() error {
	ifconfig := fmt.Sprintf("ifconfig %v %v %v mtu %d netmask %v up",
		t.Name, t.Gateway.IP, t.Gateway.IP, t.MTU, net.IP(t.Gateway.Mask),
	)
	_, ipnet, _ := net.ParseCIDR(t.Gateway.String())
	route := fmt.Sprintf("route add -net %v -interface %v",
		ipnet, t.Name,
	)
	if err := t.shell(ifconfig); err != nil {
		return logex.Trace(err)
	}
	if err := t.shell(route); err != nil {
		return logex.Trace(err)
	}
	return nil
}

func (t *Instance) Route(ipNet string) error {
	_, ipnet, err := net.ParseCIDR(ipNet)
	if err == nil {
		ipNet = ipnet.String()
	}
	return t.shell(fmt.Sprintf("route add -net %v -interface %v", ipNet, t.Name))
}
