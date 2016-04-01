package client

import (
	"fmt"
	"net"
	"time"

	"github.com/chzyer/flagly"
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
}

// -----------------------------------------------------------------------------

type ShellRouteRemove struct {
	CIDR string `name:"[0]"`
}

func (arg *ShellRouteRemove) FlaglyHandle(c *Client) error {
	if arg.CIDR == "" {
		return flagly.Error("CIDR is empty")
	}
	err := c.route.RemoveItem(arg.CIDR)
	if err != nil {
		return err
	}
	if err := c.route.Save(c.cfg.RouteFile); err != nil {
		return err
	}
	return fmt.Errorf("item '%v' removed", arg.CIDR)
}

// -----------------------------------------------------------------------------

type ShellRouteShow struct{}

func (ShellRouteShow) FlaglyHandle(c *Client, rl *readline.Instance) error {
	eis := c.route.GetEphemeralItems()
	if len(eis) > 0 {
		fmt.Fprintln(rl, "EphemeralItem:")
		for _, ei := range eis {
			fmt.Fprintf(rl, "\t%v:\t%v\t\t%v\n", ei.Expired, ei.CIDR, ei.Comment)
		}

	}
	items := c.route.GetItems()

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
	Host     string        `name:"[0]"`
}

func (arg *ShellRouteAddDomain) FlaglyDesc() string {
	return "add a route by domain with duration"
}

func (arg *ShellRouteAddDomain) FlaglyHandle(c *Client, rl *readline.Instance) error {
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
	CIDR    string `name:"[0]"`
	Comment string `name:"[1]"`
}

func (arg *ShellRouteAdd) FlaglyHandle(c *Client) (err error) {
	if arg.CIDR == "" {
		return flagly.Error("CIDR is empty")
	}
	if !arg.Force && arg.Comment == "" && arg.Duration == 0 {
		return flagly.Error("comment is empty")
	}
	if arg.Duration == 0 {
		err = c.route.AddItem(route.NewItem(arg.CIDR, arg.Comment))
		if err != nil {
			return err
		}
		err = c.route.Save(c.cfg.RouteFile)
		if err != nil {
			return err
		}
		return fmt.Errorf("route item '%v' added", arg.CIDR)
	} else {
		ei := &route.EphemeralItem{
			Item:    route.NewItem(arg.CIDR, arg.Comment),
			Expired: time.Now().Add(arg.Duration).Round(time.Second),
		}
		err = c.route.AddEphemeralItem(ei)
		if err != nil {
			return err
		}
		err = c.route.Save(c.cfg.RouteFile)
		if err != nil {
			return err
		}
		return fmt.Errorf("ephemeral item '%v' added, expired in: %v",
			ei.CIDR, ei.Expired,
		)
	}
}
