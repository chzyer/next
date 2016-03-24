package route

import "fmt"

func genAddRouteCmd(devName, cidr string) string {
	return fmt.Sprintf(
		"ip route add %v dev %v",
		formatCIDR(cidr), devName,
	)
}

func genRemoveRouteCmd(cidr string) string {
	return fmt.Sprintf("ip route delete %v", formatCIDR(cidr))
}
