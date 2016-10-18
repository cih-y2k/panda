package panda

import (
	"net"
)

// Default TODO:
var Default *Server

func init() {
	Default = NewServer(NewEngine())
}

// Server TODO:
type Server struct {
	NamespaceAPI
	engine *Engine
	ln     net.Listener
	///TODO: timeouts, max connections and so on...
}

// NewServer TODO:
func NewServer(engine *Engine) *Server {
	s := &Server{
		engine:       engine,
		NamespaceAPI: engine.ns,
	}
	return s
}

// Serve serves the incoming panda connections,
// it starts a new goroutine for each of the panda connections.
// Receives any net.Listen to listen on
// returns an error if something bad happened,
// it's a blocking func
func Serve(ln net.Listener) error {
	return Default.Serve(ln)
}

// Serve serves the incoming panda connections,
// it starts a new goroutine for each of the panda connections.
// Receives any net.Listen to listen on
// returns an error if something bad happened,
// it's a blocking func
func (s *Server) Serve(ln net.Listener) error {
	s.ln = ln
	defer s.ln.Close()

	// prepare built'n ack response server-only handler
	s.Handle("panda_ack", func(req *Request) {
		// the ack will return the connection id which will be setted on the client side in order to be synchronized with the server's
		// this id is not changed
		// after this method the client can be served(this is done on client.go)
		req.Result(int(req.Conn().ID()))
	})

	for {
		netConn, err := s.ln.Accept()
		if err != nil {
			s.engine.logf("Connection error: %s\n", err)
			continue
		}

		go func() {
			c := s.engine.acquireConn(netConn)
			c.serve() // serve blocks until error
			s.engine.releaseConn(c)
		}()
	}
}

// ListenAndServe announces on the local network address laddr.
// The network must be a stream-oriented network: "tcp", "tcp4",
// "tcp6", "unix" or "unixpacket".
// If should need to change the network use the Serve method which accepts any net.Listener
// returns an error if something bad happened,
// it's a blocking func
func ListenAndServe(network string, laddr string) error {
	return Default.ListenAndServe(network, laddr)
}

// packet-oriented network: "udp", "udp4",
// "udp6", "ip", "ip4", "ip6" or "unixgram".
// var packetNetworks = []string{"udp", "udp4", "udp6", "ip", "ip4", "ip6", "unixgram"}

// ListenAndServe announces on the local network address laddr.
// The network must be a stream-oriented network: "tcp", "tcp4",
// "tcp6", "unix" or "unixpacket".
// If should need to change the network use the Serve method which accepts any net.Listener
// returns an error if something bad happened,
// it's a blocking func
func (s *Server) ListenAndServe(network string, laddr string) error {
	/*
		var err error
		isPacketN := false
		for _, pn := range packetNetworks {
			if network == pn {
				ln, err = net.ListenPacket(network, laddr)
				isPacketN = true
				break
			}
		}*/
	ln, err := net.Listen(network, laddr)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// GetConn returns a connection by id
// the returned Conn cannot be changed
func (s *Server) GetConn(id CID) Conn {
	return s.engine.getConn(id)
}

// Exec executes a LOCAL handler registered by this server
func (s *Server) Exec(con Conn, statement string, args Args) (interface{}, error) {
	if c, ok := con.(*conn); ok {
		req := c.acquireRequest(statement, args)
		defer c.releaseRequest(req)
		return s.engine.handlers.exec(req)
	}
	return nil, errHandlerNotFound.Format(statement)
}

// Close terminates the server
func (s *Server) Close() error {
	return s.ln.Close()
}
