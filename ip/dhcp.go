package ip

// use for alloc ip address
type DHCP struct {
	Gateway   IP
	Boardcast IP
	IpSize    int
	bitmap    []byte
}

func NewDHCP(ipnet *IPNet) *DHCP {
	gateway := ipnet.IP.Clone()
	gatewayInt := gateway.Int()

	ones, bits := ipnet.Mask.Size()
	ipSize := 1<<uint(bits-ones) -
		1 - // boardcast
		gatewayInt&((1<<uint32(bits-ones))-1) // gateway offset

	boardcast := ParseIntIP(gatewayInt + uint32(ipSize))

	bitmap := make([]byte, (ipSize+7)/8)

	dhcp := &DHCP{
		Gateway:   gateway,
		Boardcast: boardcast,
		bitmap:    bitmap,
		IpSize:    int(ipSize),
	}
	return dhcp
}

func (d *DHCP) IsExistIP(ip IP) bool {
	ipInt := ip.Int()
	gateway := d.Gateway.Int()
	boardcast := d.Boardcast.Int()
	if ipInt == gateway || ipInt == boardcast {
		return true
	} else if ipInt < gateway || ipInt > boardcast {
		return false
	} else {
		offset := ipInt - gateway - 1
		bitset := d.bitmap[offset/8] // can't panic
		return bitset&(1<<(offset&7)) > 0
	}
}

func (d *DHCP) Release(ip IP) bool {
	ipInt := ip.Int()
	gateway := d.Gateway.Int()
	boardcast := d.Boardcast.Int()
	if ipInt > gateway && ipInt < boardcast {
		offset := ipInt - gateway - 1
		idx := offset / 8
		isExist := d.bitmap[idx]&(1<<(offset&7)) > 0
		d.bitmap[idx] &= ^(1 << (offset & 7))
		return isExist
	} else {
		return false
	}
}

func (d *DHCP) Alloc() *IP {
	gateway := d.Gateway.Int() + 1
	boardcast := d.Boardcast.Int()
	for i := 0; i < len(d.bitmap); i++ {
		if d.bitmap[i] == 255 {
			continue
		}
		// we can alloc ip here
		for j := uint32(0); j < 8; j++ {
			ipInt := gateway + uint32(i*8) + j
			if ipInt >= boardcast {
				break
			}
			if d.bitmap[i]&(1<<j) == 0 {
				d.bitmap[i] |= (1 << j)
				ip := ParseIntIP(ipInt)
				return &ip
			}
		}
	}
	return nil
}
