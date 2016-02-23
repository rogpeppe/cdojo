package cdojo

import (
	"fmt"
)

// Message represents a message from the server to a client.
type Message struct {
	// Kind holds the kind of message. This determines which fields are valid.
	Kind Kind

	// Id holds the client id that the message was sent from. Valid for all messages
	// sent from the server to a client, and for the initial Connect message
	// sent from a client.
	Id string `json:",omitempty"`

	// Name holds the name that the client wants to call itself. Valid for KindConnect.
	Name string `json:",omitempty"`

	// Addr holds the address of the client. Valid for KindConnect.
	Addr string `json:",omitempty"`

	// Text holds a textual message. Valid for KindText.
	Text string `json:",omitempty"`
}

// Kind represents the kind of a message.
type Kind string

const (
	// KindUser is sent to a client for each already-connected user.
	KindUser Kind = "user"
	// KindConnect is sent when a client first connects.
	KindConnect Kind = "connect"
	// KindText is sent when a client sends some text.
	KindText Kind = "text"
	// KindDisconnect is sent when a client disconnects.
	KindDisconnect Kind = "disconnect"
)

func (m Message) String() string {
	switch m.Kind {
	case KindText:
		return fmt.Sprintf("%s[%s]: %v", m.Name, m.Id, m.Text)
	case KindConnect:
		return fmt.Sprintf("%s[%v] joined", m.Name, m.Id)
	case KindDisconnect:
		return fmt.Sprintf("%s[%v] left", m.Name, m.Id)
	case KindUser:
		return fmt.Sprintf("%s[%v] is here", m.Name, m.Id)
	default:
		return fmt.Sprintf("unknown message %q", m.Kind)
	}
}
