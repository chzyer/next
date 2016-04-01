package ip

import (
	"net"
	"net/url"
	"strings"
)

func FormatHost(host string) string {
	if strings.HasPrefix(host, "http") {
		u, err := url.Parse(host)
		if err == nil {
			host = u.Host
		}
	}
	return host
}

func LookupHost(host string) ([]string, error) {
	return net.LookupHost(FormatHost(host))
}
