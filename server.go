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
	engine                *Engine
	ln                    net.Listener
	connectedEvtListeners []func(*Conn) ///TODO: na kanw kai ta onClose enoeite kai diagrafw ta connections sto engine dn xreiazonte...
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

func (s *Server) serve(ln net.Listener) error {
	s.ln = ln

	for {
		netConn, err := s.ln.Accept()
		if err != nil {
			s.engine.logf("Connection error: %s\n", err)
			continue
		}

		go func() {
			c := s.engine.acquireConn(netConn)
			go s.emitConnected(c)
			/// TODO: edw na kanw usage KAPWS to emitConnected gia na borw na elenxw to connected an kanw .serve i an kanw ena http serve..
			c.serve() // serve blocks until error
			s.engine.releaseConn(c)
		}()
	}
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
		req.Result(int(req.Conn.ID()))
	})

	for {
		netConn, err := s.ln.Accept()
		if err != nil {
			s.engine.logf("Connection error: %s\n", err)
			continue
		}

		go func() {
			c := s.engine.acquireConn(netConn)
			go s.emitConnected(c)
			c.serve() // serve blocks until connection stop reading
			s.engine.releaseConn(c)
		}()
	}
}

// OnConnection registers an event callback and raise (them) when a new connection has been connected to the server
// it has no many usages because Conn is limited for your own safety, but you can do some useful staff, like
// adding this connection to a map or slice and check if a connection asking for something passed its custom verification
// although you can do that with a middleware, it's useful to have an event like this for any case
//
// Note: Register OnConnection event callbacks before you started the server, otherwise you may experience some issues
func (s *Server) OnConnection(cb func(*Conn)) {
	s.connectedEvtListeners = append(s.connectedEvtListeners, cb)
}

// emitConnected runs in each own goroutine before serve in order to be ready to `.Do` to the client for example
func (s *Server) emitConnected(c *Conn) {
	for _, l := range s.connectedEvtListeners {
		l(c)
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

// Exec executes a LOCAL handler registered by this server
func (s *Server) Exec(c *Conn, statement string, args Args) (interface{}, error) {
	req := c.acquireRequest(statement, args)
	defer c.releaseRequest(req)
	return s.engine.handlers.exec(req)
}

// Close terminates the server
func (s *Server) Close() error {
	return s.ln.Close()
}
