package dchan

import "fmt"

func CheckType(name string) error {
	if GetChannelType(name) == nil {
		return fmt.Errorf("invalid channel type: %v", name)
	}
	return nil
}

func GetChannelType(name string) ChannelFactory {
	switch name {
	case "http":
		return HttpChanFactory{}
	case "tcp":
		return TcpChanFactory{}
	case "udp":
		return NewUdpChanFactory()
	default:
		return nil
	}
}
