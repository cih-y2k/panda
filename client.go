package panda

import (
	"net"
)

// Client TODO:
type Client struct {
	*ClientNamespace
	engine *Engine
}

// NewClient TODO:
func NewClient(engine *Engine) *Client {
	return &Client{
		engine: engine,
		ClientNamespace: &ClientNamespace{
			namespace: engine.ns,
		}}
}

// Connect TODO:
// non- blocking func
func (s *Client) Connect(laddr string) error {
	return s.Dial("tcp4", laddr)
}

// Dial connects to the address on the named network.
//
// Known networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only),
// "udp", "udp4" (IPv4-only), "udp6" (IPv6-only), "ip", "ip4"
// (IPv4-only), "ip6" (IPv6-only), "unix", "unixgram" and
// "unixpacket".
//
// For TCP and UDP networks, addresses have the form host:port.
// If host is a literal IPv6 address it must be enclosed
// in square brackets as in "[::1]:80" or "[ipv6-host%zone]:80".
// The functions JoinHostPort and SplitHostPort manipulate addresses
// in this form.
// If the host is empty, as in ":80", the local system is assumed.
//
// Examples:
//	Dial("tcp", "192.0.2.1:80")
//	Dial("tcp", "golang.org:http")
//	Dial("tcp", "[2001:db8::1]:http")
//	Dial("tcp", "[fe80::1%lo0]:80")
//	Dial("tcp", ":80")
//
// For IP networks, the network must be "ip", "ip4" or "ip6" followed
// by a colon and a protocol number or name and the addr must be a
// literal IP address.
//
// Examples:
//	Dial("ip4:1", "192.0.2.1")
//	Dial("ip6:ipv6-icmp", "2001:db8::1")
//
// For Unix networks, the address must be a file system path.
func (s *Client) Dial(network string, laddr string) error {
	netConn, err := net.Dial(network, laddr)
	if err != nil {
		return err
	}
	s.conn = s.engine.acquireConn(netConn) // conn lives in the ClientNamespace

	go func() {
		s.conn.serve() // serve blocks until returns
		s.engine.releaseConn(s.conn)
	}()
	return s.ack(netConn)
}

func (s *Client) ack(netConn net.Conn) error {
	// wait for ack before serving
	res, err := s.Do("panda_ack")

	if err != nil {
		s.engine.logf("Error on panda ack when receiving the connection's ID, error: %s", err)
		return err
	}
	// some decoders/used for deserialization see the int as float64(standar json), some other like godec package see int as uint64, so :

	// sync the id from the server side
	id, err := DecodeInt(res)
	if err != nil {
		panic(err)
	}

	s.conn.setID(id)
	return nil
}

// Conn returns the client's connection
func (s *Client) Conn() Conn {
	return s.conn
}

// Exec executes a LOCAL handler registered by this client
func (s *Client) Exec(statement string, args ...Arg) (interface{}, error) {
	return s.engine.handlers.exec(s.conn, statement, args...)
}

// Close terminates the underline connection (net.Conn)
func (s *Client) Close() error {
	return s.conn.Close()
}
