package route

import "fmt"

func genAddRouteCmd(devName, cidr string) string {
	return fmt.Sprintf(
		"route add -net %v -interface %v",
		FormatCIDR(cidr), devName,
	)
}

func genRemoveRouteCmd(cidr string) string {
	return fmt.Sprintf("route delete -net %v", FormatCIDR(cidr))
}
