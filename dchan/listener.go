package dchan

import (
	"net"
	"strconv"
	"strings"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
)

// add self monitor
type Listener struct {
	ln          net.Listener
	flow        *flow.Flow
	delegate    SvrDelegate
	chanFactory ChannelFactory
	port        int
	onClose     func()
}

func NewListener(f *flow.Flow, d SvrDelegate, chanTyp string, c func()) (*Listener, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	addr := ln.Addr().String()
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		addr = addr[idx+1:]
	}
	port, err := strconv.Atoi(addr)
	if err != nil {
		panic(err)
	}
	dcln := &Listener{
		ln:          ln,
		port:        port,
		delegate:    d,
		onClose:     c,
		chanFactory: GetChannelType(chanTyp),
	}
	f.ForkTo(&dcln.flow, dcln.Close)
	return dcln, nil
}

func (d *Listener) GetPort() int {
	return d.port
}

type listenerDelegate struct {
	delegate SvrDelegate
}

func (d *listenerDelegate) Init(userId int) (toUser chan<- *packet.Packet, err error) {
	_, toUser, err = d.delegate.GetUserChannelFromDataChannel(userId)
	if err != nil {
		return nil, err
	}
	return toUser, nil
}

func (d *listenerDelegate) OnInited(ch Channel) {
	d.delegate.OnNewChannel(ch)
}

func (d *Listener) Accept() (Channel, error) {
	conn, err := d.ln.Accept()
	if err != nil {
		return nil, logex.Trace(err)
	}

	session := packet.NewSessionSvr(d.delegate)
	delegate := &listenerDelegate{d.delegate}
	ch := d.chanFactory.NewServer(d.flow, session, conn, delegate)
	return ch, nil
}

func (d *Listener) Serve() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	for !d.flow.IsClosed() {
		ch, err := d.Accept()
		if err != nil {
			break
		}
		go ch.Run()
	}
}

func (d *Listener) Close() {
	if !d.flow.MarkExit() {
		return
	}
	logex.Info("listener:", d.port, "closed")
	d.ln.Close()
	d.flow.Close()
	d.onClose()
}
