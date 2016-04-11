package clish

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/chzyer/flagly"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/route"
	"github.com/chzyer/next/util"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type ShellRoute struct {
	Add       *ShellRouteAdd       `flagly:"handler"`
	AddDomain *ShellRouteAddDomain `flagly:"handler"`
	Show      *ShellRouteShow      `flagly:"handler"`
	Remove    *ShellRouteRemove    `flagly:"handler"`
	Get       *ShellRouteGet       `flagly:"handler"`
}

// -----------------------------------------------------------------------------

type ShellRouteRemove struct {
	CIDR string `type:"[0]"`
}

func (arg *ShellRouteRemove) FlaglyHandle(c Client) error {
	if arg.CIDR == "" {
		return flagly.Error("CIDR is empty")
	}
	ch, err := c.GetRoute()
	if err != nil {
		return err
	}
	if err := ch.RemoveItem(arg.CIDR); err != nil {
		return err
	}
	if err := c.SaveRoute(); err != nil {
		return err
	}
	return fmt.Errorf("item '%v' removed", arg.CIDR)
}

// -----------------------------------------------------------------------------

type ShellRouteShow struct{}

func (ShellRouteShow) FlaglyHandle(c Client, rl *readline.Instance) error {
	route, err := c.GetRoute()
	if err != nil {
		return err
	}
	eis := route.GetEphemeralItems()
	if len(eis) > 0 {
		fmt.Fprintln(rl, "EphemeralItem:")
		for _, ei := range eis {
			fmt.Fprintf(rl, "\t%v:\t%v\t\t%v\n", ei.Expired, ei.CIDR, ei.Comment)
		}

	}
	items := route.GetItems()

	if len(items) > 0 {
		if len(eis) > 0 {
			fmt.Fprintln(rl)
		}
		max := 0
		for _, item := range items {
			if len(item.CIDR) > max {
				max = len(item.CIDR)
			}
		}

		fmt.Fprintln(rl, "Item:")
		for _, item := range items {
			fmt.Fprintf(rl, "\t%v\t%v\n",
				util.FillString(item.CIDR, max, " "), item.Comment,
			)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

type ShellRouteAddDomain struct {
	Duration time.Duration `name:"d" desc:"ephemeral node duration time" default:"6h"`
	Host     string        `type:"[0]"`
}

func (arg *ShellRouteAddDomain) FlaglyDesc() string {
	return "add a route by domain with duration"
}

func (arg *ShellRouteAddDomain) FlaglyHandle(c Client, rl *readline.Instance) error {
	if arg.Host == "" {
		return flagly.Error("host is required")
	}
	if arg.Duration == 0 {
		arg.Duration = 6 * time.Hour
	}
	ips, err := net.LookupIP(arg.Host)
	if err != nil {
		return err
	}
	cfg := &ShellRouteAdd{
		Duration: arg.Duration,
	}
	for _, ip := range ips {
		cfg.CIDR = ip.String()
		if err := cfg.FlaglyHandle(c); err != nil {
			if logex.Equal(route.ErrRouteItemExists, err) {
				fmt.Fprintf(rl, "ip %v is exists! ignore\n", cfg.CIDR)
				continue
			}
		}
		fmt.Fprintf(rl, "ip %v is added!\n", cfg.CIDR)
	}
	return nil
}

// -----------------------------------------------------------------------------

type ShellRouteAdd struct {
	Duration time.Duration `name:"d" desc:"ephemeral node duration time"`

	Force   bool   `name:"f" desc:"force execute even comment is missing"`
	CIDR    string `type:"[0]"`
	Comment string `type:"[1]"`
}

func (arg *ShellRouteAdd) FlaglyHandle(c Client) (err error) {
	if arg.CIDR == "" {
		return flagly.Error("CIDR is empty")
	}
	if !arg.Force && arg.Comment == "" && arg.Duration == 0 {
		return flagly.Error("comment is empty")
	}
	if arg.Duration == 0 {
		item, err := route.NewItemCIDR(arg.CIDR, arg.Comment)
		if err != nil {
			return flagly.Error(err.Error())
		}
		routeTable, err := c.GetRoute()
		if err != nil {
			return err
		}
		err = routeTable.AddItem(item)
		if err != nil {
			return err
		}
		err = c.SaveRoute()
		if err != nil {
			return err
		}
		return fmt.Errorf("route item '%v' added", arg.CIDR)
	} else {
		item, err := route.NewItemCIDR(arg.CIDR, arg.Comment)
		if err != nil {
			return err
		}
		ei := &route.EphemeralItem{
			Item:    item,
			Expired: time.Now().Add(arg.Duration).Round(time.Second),
		}
		routeTable, err := c.GetRoute()
		if err != nil {
			return err
		}
		err = routeTable.AddEphemeralItem(ei)
		if err != nil {
			return err
		}
		err = c.SaveRoute()
		if err != nil {
			return err
		}
		return fmt.Errorf("ephemeral item '%v' added, expired in: %v",
			ei.CIDR, ei.Expired,
		)
	}
}

// -----------------------------------------------------------------------------
type ShellRouteGet struct {
	Host string `type:"[0]" name:"ip/host"`
}

func (s *ShellRouteGet) FlaglyHandle(c Client) error {
	if s.Host == "" {
		return flagly.Error("Host is required")
	}
	cidrs := []string{route.FormatCIDR(s.Host)}

	if !ip.IsIP(cidrs[0]) {
		ips, err := ip.LookupHost(s.Host)
		if err != nil {
			return err
		}
		cidrs = ips
	}
	max := 0
	for idx := range cidrs {
		cidrs[idx] = route.FormatCIDR(cidrs[idx])
		if len(cidrs[idx]) > max {
			max = len(cidrs[idx])
		}
	}

	buf := bytes.NewBuffer(nil)
	for _, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		routeTable, err := c.GetRoute()
		if err != nil {
			return err
		}
		item := routeTable.Match(ipnet)
		buf.WriteString(util.FillString(cidr, max, " ") + "    ")
		if item != nil {
			buf.WriteString("ok\n")
		} else {
			buf.WriteString("missing\n")
		}
	}

	return fmt.Errorf(strings.TrimSpace(buf.String()))
}
