package tunnel

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"gopkg.in/logex.v1"
)

const (
	cIFF_TUN   = 0x0001
	cIFF_NO_PI = 0x1000
)

func OpenTun(idx int) (*os.File, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, logex.Trace(err)
	}
	_, err = createInterface(file.Fd(), fmt.Sprintf("utun%d", idx), cIFF_TUN|cIFF_NO_PI)
	if err != nil {
		return nil, logex.Trace(err)
	}
	return file, nil
}

func createInterface(fd uintptr, ifName string, flags uint16) (createdIFName string, err error) {
	var req ifReq
	req.Flags = flags
	copy(req.Name[:], ifName)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		err = errno
		return
	}
	createdIFName = strings.Trim(string(req.Name[:]), "\x00")
	return
}

func (t *Instance) setupTun() error {
	dev := fmt.Sprintf("ip link set dev %v up mtu %v qlen 100",
		t.Name, t.MTU,
	)
	devAddr := fmt.Sprintf("ip addr add dev %v local %v peer %v",
		t.Name, t.Config.Gateway.IP, t.Config.Gateway.IP,
	)
	_, net, err := net.ParseCIDR((t.Gateway).String())
	if err != nil {
		return logex.Trace(err)
	}
	route := fmt.Sprintf("ip route add %v via %v dev %v",
		net, t.Config.Gateway.IP, t.Name,
	)
	if err := t.shell(dev); err != nil {
		return logex.Trace(err)
	}

	if err := t.shell(devAddr); err != nil {
		return logex.Trace(err)
	}

	if err := t.shell(route); err != nil {
		return logex.Trace(err)
	}

	return nil
}

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}
