package socket

import (
	"errors"
	"io/fs"
	"net"

	"github.com/wnxd/microdbg/filesystem"
)

type Network = string

const (
	TCP        Network = "tcp"
	TCP4       Network = "tcp4"
	TCP6       Network = "tcp6"
	UDP        Network = "udp"
	UDP4       Network = "udp4"
	UDP6       Network = "udp6"
	IP         Network = "ip"
	IP4        Network = "ip4"
	IP6        Network = "ip6"
	Unix       Network = "unix"
	UnixGram   Network = "unixgram"
	UnixPacket Network = "unixpacket"
)

type Socket interface {
	Server
	Client
	Conn
}

type Server interface {
	Bind(addr string) error
	Listen() error
	Accept() (Conn, error)
}

type Client interface {
	Connect(addr string) error
}

type Conn interface {
	filesystem.ControlFile
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
}

type socket struct {
	t Network
	a string
	l net.Listener
	c net.Conn
}

func New(network Network) Socket {
	return &socket{t: network}
}

func (s *socket) Bind(addr string) error {
	if s.a != "" {
		return ErrAlreadyBind
	} else if s.c != nil {
		return ErrAlreadyConnect
	}
	s.a = addr
	return nil
}

func (s *socket) Listen() error {
	if s.a == "" {
		return ErrNotBind
	} else if s.l != nil {
		return ErrAlreadyListen
	} else if s.c != nil {
		return ErrAlreadyConnect
	}
	var err error
	s.l, err = net.Listen(s.t, s.a)
	return err
}

func (s *socket) Accept() (Conn, error) {
	if s.l == nil {
		return nil, ErrNotListen
	} else if s.c != nil {
		return nil, ErrAlreadyConnect
	}
	conn, err := s.l.Accept()
	if err != nil {
		return nil, err
	}
	return &socket{t: s.t, c: conn}, nil
}

func (s *socket) Connect(addr string) error {
	if s.a != "" {
		return ErrAlreadyBind
	} else if s.l != nil {
		return ErrAlreadyListen
	} else if s.c != nil {
		return ErrAlreadyConnect
	}
	var err error
	s.c, err = net.Dial(s.t, addr)
	return err
}

func (s *socket) Close() error {
	if s.l != nil {
		return s.l.Close()
	} else if s.c != nil {
		return s.c.Close()
	}
	return nil
}

func (s *socket) Stat() (fs.FileInfo, error) {
	return info{}, nil
}

func (s *socket) Read(b []byte) (int, error) {
	if s.c == nil {
		return 0, ErrNotConnect
	}
	return s.c.Read(b)
}

func (s *socket) Write(b []byte) (int, error) {
	if s.c == nil {
		return 0, ErrNotConnect
	}
	return s.c.Write(b)
}

func (s *socket) Control(op int, arg any) error {
	return errors.ErrUnsupported
}
