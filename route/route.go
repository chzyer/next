package route

import (
	"net"
	"strings"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/util"
	"gopkg.in/logex.v1"
)

var (
	ErrRouteItemNotFound = logex.Define("route item '%v' not found")
	ErrRouteItemExists   = logex.Define("route item '%v' is exists")
)

type Item struct {
	CIDR    string
	Comment string
}

func NewItem(cidr string, comment string) *Item {
	return &Item{formatCIDR(cidr), comment}
}

type Route struct {
	flow             *flow.Flow
	items            *Items
	ephemeralItems   *EphemeralItems
	devName          string
	newEphemeralItem chan struct{}
}

func NewRoute(f *flow.Flow, devName string) *Route {
	r := &Route{
		flow:             f,
		devName:          devName,
		items:            &Items{},
		ephemeralItems:   NewEphemeralItems(),
		newEphemeralItem: make(chan struct{}, 1),
	}
	go r.loop()
	return r
}

func (r *Route) GetEphemeralItems() []EphemeralItem {
	ret := make([]EphemeralItem, 0, r.ephemeralItems.Len())
	for elem := r.ephemeralItems.list.Front(); elem != nil; elem = elem.Next() {
		ei := elem.Value.(*EphemeralItem)
		ret = append(ret, *ei)
	}
	return ret
}

func (r *Route) GetItems() Items {
	return *r.items
}

func (r *Route) loop() {
loop:
	for {
		i := r.ephemeralItems.GetFront()
		if i == nil {
			select {
			case <-r.newEphemeralItem:
			case <-r.flow.IsClose():
				break loop
			}
		} else {
			now := time.Now()
			if now.After(i.Expired) {
				logex.Infof("route '%v' is expired", i.CIDR)
				err := r.RemoveEphemeralItem(i.CIDR)
				if err != nil {
					logex.Error("remove route item fail:", err.Error())
				}
			} else {
				select {
				case <-time.After(i.Expired.Sub(now)):
				case <-r.newEphemeralItem:
				case <-r.flow.IsClose():
					break loop
				}
			}
		}
	}
}

func (r *Route) RemoveItem(cidr string) error {
	if item := r.items.Remove(cidr); item != nil {
		return r.DeleteRoute(cidr)
	}
	return ErrRouteItemNotFound.Format(cidr)
}

func (r *Route) RemoveEphemeralItem(cidr string) error {
	if r.ephemeralItems.Remove(cidr) != nil {
		return logex.Trace(r.DeleteRoute(cidr))
	}
	return ErrRouteItemNotFound.Format(cidr)
}

func (r *Route) PersistEphemeralItem(cidr string) error {
	if ei := r.ephemeralItems.Remove(cidr); ei != nil {
		r.items.Append(ei.Item)
		r.items.Sort()
		return nil
	}
	return ErrRouteItemNotFound.Format(cidr)
}

func (r *Route) AddEphemeralItem(i *EphemeralItem) error {
	r.ephemeralItems.Add(i)
	select {
	case r.newEphemeralItem <- struct{}{}:
	default:
	}
	return logex.Trace(r.SetRoute(i.CIDR))
}

func (r *Route) AddItem(i *Item) error {
	i.CIDR = formatCIDR(i.CIDR)
	if err := r.items.Append(i); err != nil {
		return err
	}
	r.items.Sort()
	return logex.Trace(r.SetRoute(i.CIDR))
}

func (r *Route) DeleteRoute(cidr string) error {
	sh := genRemoveRouteCmd(cidr)
	return logex.Trace(util.Shell(sh))
}

func (r *Route) SetRoute(cidr string) error {
	sh := genAddRouteCmd(r.devName, cidr)
	return logex.Trace(util.Shell(sh))
}

func (r *Route) Load(fp string) error {
	return nil
}

func (r *Route) Save(fp string) error {
	return nil
}

func formatCIDR(cidr string) string {
	if idx := strings.Index(cidr, "/"); idx < 0 {
		cidr += "/32"
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err == nil {
		cidr = ipnet.String()
	}

	return cidr
}
