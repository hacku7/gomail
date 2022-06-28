package send

import (
	"fmt"
	"github.com/hacku7/gomail/msg"
	"io"
)

// Sender is the interface that wraps the Send method.
//
// Send sends an email to the given addresses.
type Sender interface {
	Send(from string, to []string, msg io.WriterTo) error
}

// SendCloser is the interface that groups the Send and Close methods.
type SendCloser interface {
	Sender
	Close() error
}

// A SendFunc is a function that sends emails to the given addresses.
//
// The SendFunc type is an adapter to allow the use of ordinary functions as
// email senders. If f is a function with the appropriate signature, SendFunc(f)
// is a Sender object that calls f.
type SendFunc func(from string, to []string, msg io.WriterTo) error

// Send calls f(from, to, msg).
func (f SendFunc) Send(from string, to []string, msg io.WriterTo) error {
	return f(from, to, msg)
}

// Send sends emails using the given Sender.
func Send(s Sender, msg ...*msg.Message) error {
	for i, m := range msg {
		if err := send(s, m); err != nil {
			return fmt.Errorf("gomail: could not send email %d: %v", i+1, err)
		}
	}

	return nil
}

func send(s Sender, m *msg.Message) error {
	from, err := m.GetFrom()
	if err != nil {
		return err
	}

	to, err := m.GetRecipients()
	if err != nil {
		return err
	}

	if err := s.Send(from, to, m); err != nil {
		return err
	}

	return nil
}
