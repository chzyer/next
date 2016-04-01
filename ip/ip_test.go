package ip

import (
	"net"
	"testing"

	"github.com/chzyer/next/test"
)

func TestMatchIPNet(t *testing.T) {
	defer test.New(t)

	var result = []struct {
		Child  string
		Parent string
		Match  bool
	}{
		{"10.0.0.0/24", "10.0.0.0/16", true},
		{"10.0.0.0/24", "10.0.0.0/24", true},
		{"10.0.0.1/16", "10.0.0.0/24", false},
		{"1.2.3.4/32", "1.2.3.0/24", true},
		{"1.2.4.4/24", "1.2.3.0/24", false},
	}

	for _, r := range result {
		_, child, err := net.ParseCIDR(r.Child)
		test.Nil(err)
		_, parent, err := net.ParseCIDR(r.Parent)
		test.Nil(err)
		test.Equal(MatchIPNet(child, parent), r.Match)
	}
}
