package cdojo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/rogpeppe/fastuuid"
	"io"
	"log"
	"net"
	"sync"
)

// Server represents a message server.
type Server struct {
	lis     net.Listener
	mu      sync.Mutex
	clients map[*serverClient]struct{}
}

type serverClient struct {
	srv    *Server
	conn   net.Conn
	closed bool
	send   chan Message
	id     string
	addr   string

	mu    sync.Mutex
	name_ string
}

// NewServer returns a new server listening on the
// given TCP address (e.g. ":1234").
func NewServer(addr string) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		lis:     lis,
		clients: make(map[*serverClient]struct{}),
	}
	go srv.run()
	return srv, nil
}

func (srv *Server) run() {
	for {
		conn, err := srv.lis.Accept()
		if err != nil {
			log.Fatalf("accept failed: %v", err)
		}
		srv.startClient(conn)
	}
}

func (srv *Server) startClient(conn net.Conn) {
	client := &serverClient{
		conn: conn,
		srv:  srv,
		send: make(chan Message, 100),
		id:   newId(),
		addr: conn.RemoteAddr().String(),
	}
	srv.mu.Lock()
	joinMessages := make([]Message, len(srv.clients))
	for c := range srv.clients {
		joinMessages = append(joinMessages, Message{
			Kind: KindUser,
			Id:   c.id,
			Name: c.name(),
			Addr: c.addr,
		})
	}
	srv.clients[client] = struct{}{}
	srv.mu.Unlock()
	go client.writer()
	for _, m := range joinMessages {
		client.send <- m
	}
	go client.reader()
}

func (c *serverClient) reader() {
	addr := c.conn.RemoteAddr().String()
	defer c.close()
	dec := json.NewDecoder(c.conn)
	var initialMessage Message
	if err := dec.Decode(&initialMessage); err != nil {
		log.Printf("bad initial message from %s", addr)
		return
	}
	if initialMessage.Kind != KindConnect {
		log.Printf("bad initial message kind from %s (got %s, want %s)", addr, initialMessage.Kind, KindConnect)
		return
	}
	name := initialMessage.Name
	if name == "" {
		name = c.id
	}
	c.setName(name)
	c.sendAll(Message{
		Kind: KindConnect,
		Name: name,
		Addr: addr,
		Id:   c.id,
	})
	for {
		var m Message
		err := dec.Decode(&m)
		if err != nil {
			if err != io.EOF {
				log.Printf("error from %v (%v; %v): %v", addr, name, c.id, err)
				return
			}
			return
		}
		if m.Kind != KindText {
			log.Printf("ignoring message kind from %v (%v; %v): %v", addr, name, c.id, m.Kind)
			continue
		}
		c.sendAll(Message{
			Kind: KindText,
			Id:   c.id,
			Name: name,
			Text: m.Text,
		})
	}
	c.sendAll(Message{
		Kind: KindDisconnect,
		Id:   c.id,
		Name: name,
	})
}

func (c *serverClient) setName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.name_ = name
}

func (c *serverClient) name() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.name_
}

func (c *serverClient) sendAll(msg Message) {
	c.srv.sendAll(nil, msg)
}

func (srv *Server) sendAll(client *serverClient, msg Message) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	for c := range srv.clients {
		if c == client {
			continue
		}
		select {
		case c.send <- msg:
		default:
			log.Printf("discarded message to %v", c.conn.RemoteAddr())
		}
	}
}

// SendAll sends a message to all currently connected clients.
func (srv *Server) SendAll(msg Message) {
	srv.sendAll(nil, msg)
}

const maxMessageSize = 128 * 1024

func (c *serverClient) writer() {
	w := bufio.NewWriterSize(c.conn, maxMessageSize)
	enc := json.NewEncoder(w)
	defer c.close()
	for m := range c.send {
		for m.Kind != "" {
			if err := enc.Encode(m); err != nil {
				return
			}
			var ok bool
			select {
			case m, ok = <-c.send:
				if !ok {
					return
				}
			default:
				m.Kind = ""
			}
		}
		if err := w.Flush(); err != nil {
			return
		}
	}
}

func (c *serverClient) close() {
	c.srv.mu.Lock()
	defer c.srv.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.send)
	c.conn.Close()
	delete(c.srv.clients, c)
}

var idGen = fastuuid.MustNewGenerator()

func newId() string {
	id := idGen.Next()
	return fmt.Sprintf("%x", id[0:16])
}
