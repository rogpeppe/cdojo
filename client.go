package cdojo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
)

type Client struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
	msgc chan Message
}

func NewClient(addr string, name string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &Client{
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
		msgc: make(chan Message),
	}
	c.sendMessage(Message{
		Kind: KindConnect,
		Name: name,
	})
	go c.readMessages()

	return c, nil
}

func (c *Client) Messages() <-chan Message {
	return c.msgc
}

func (c *Client) readMessages() {
	defer close(c.msgc)
	for {
		m, err := c.read()
		if err != nil {
			if err != io.EOF {
				log.Printf("client read error: %v", err)
			}
			return
		}
		c.msgc <- m
	}
}

func (c *Client) read() (Message, error) {
	var m Message
	if err := c.dec.Decode(&m); err != nil {
		return Message{}, fmt.Errorf("cannot decode message: %v", err)
	}
	return m, nil
}

func (c *Client) SendText(text string) error {
	return c.sendMessage(Message{
		Kind: KindText,
		Text: text,
	})
}

func (c *Client) sendMessage(m Message) error {
	if err := c.enc.Encode(m); err != nil {
		return fmt.Errorf("cannot send message: %v", err)
	}
	return nil
}
