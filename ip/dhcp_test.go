package ip

import (
	"testing"

	"github.com/chzyer/test"
)

func TestDHCP(t *testing.T) {
	defer test.New(t)
	ipnet, err := ParseCIDR("10.6.0.1/24")
	test.Nil(err)
	d := NewDHCP(ipnet)
	first := d.Alloc()
	test.NotNil(first)
	test.Equal(*first, ParseIP("10.6.0.2"))
	test.Should(d.IsExistIP(*first))
	test.Should(!d.IsExistIP(ParseIP("10.6.0.3")))
	test.Should(!d.IsExistIP(ParseIP("10.6.0.10")))
	second := d.Alloc()
	test.NotNil(second)
	test.Equal(*second, ParseIP("10.6.0.3"))

	test.Should(d.Release(*first))
	test.Should(!d.Release(ParseIP("10.6.0.10")))
	test.Should(!d.IsExistIP(*first))

	first2 := d.Alloc()
	test.NotNil(first2)
	test.Equal(*first2, *first)
	d.Release(*first)
	d.Release(*second)

	for i := 0; i < d.IpSize; i++ {
		test.NotNil(d.Alloc())
	}
	// must be full
	test.Nil(d.Alloc())
}
