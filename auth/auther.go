package auth

import (
	"bytes"
	"errors"
	"fmt"
	"net/smtp"
)

// LoginAuth is a smtp.Auth that implements the LOGIN authentication mechanism.
type LoginAuth struct {
	Username string
	Password string
	Host     string
}

func (a *LoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		advertised := false
		for _, mechanism := range server.Auth {
			if mechanism == "LOGIN" {
				advertised = true
				break
			}
		}
		if !advertised {
			return "", nil, errors.New("gomail: unencrypted connection")
		}
	}
	if server.Name != a.Host {
		return "", nil, errors.New("gomail: wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *LoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	switch {
	case bytes.Equal(fromServer, []byte("Username:")):
		return []byte(a.Username), nil
	case bytes.Equal(fromServer, []byte("Password:")):
		return []byte(a.Password), nil
	default:
		return nil, fmt.Errorf("gomail: unexpected server challenge: %s", fromServer)
	}
}
