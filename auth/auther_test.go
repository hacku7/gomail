package auth

import (
	"net/smtp"
	"testing"
)

const (
	TestUser = "user"
	TestPwd  = "pwd"
	TestHost = "smtp.example.com"
)

type AuthTest struct {
	auths      []string
	challenges []string
	tls        bool
	wantData   []string
	wantError  bool
}

func TestNoAdvertisement(t *testing.T) {
	testLoginAuth(t, &AuthTest{
		auths:     []string{},
		tls:       false,
		wantError: true,
	})
}

func TestNoAdvertisementTLS(t *testing.T) {
	testLoginAuth(t, &AuthTest{
		auths:      []string{},
		challenges: []string{"Username:", "Password:"},
		tls:        true,
		wantData:   []string{"", TestUser, TestPwd},
	})
}

func TestLogin(t *testing.T) {
	testLoginAuth(t, &AuthTest{
		auths:      []string{"PLAIN", "LOGIN"},
		challenges: []string{"Username:", "Password:"},
		tls:        false,
		wantData:   []string{"", TestUser, TestPwd},
	})
}

func TestLoginTLS(t *testing.T) {
	testLoginAuth(t, &AuthTest{
		auths:      []string{"LOGIN"},
		challenges: []string{"Username:", "Password:"},
		tls:        true,
		wantData:   []string{"", TestUser, TestPwd},
	})
}

func testLoginAuth(t *testing.T, test *AuthTest) {
	auth := &LoginAuth{
		Username: TestUser,
		Password: TestPwd,
		Host:     TestHost,
	}
	server := &smtp.ServerInfo{
		Name: TestHost,
		TLS:  test.tls,
		Auth: test.auths,
	}
	proto, toServer, err := auth.Start(server)
	if err != nil && !test.wantError {
		t.Fatalf("loginAuth.Start(): %v", err)
	}
	if err != nil && test.wantError {
		return
	}
	if proto != "LOGIN" {
		t.Errorf("invalid protocol, got %q, want LOGIN", proto)
	}

	i := 0
	got := string(toServer)
	if got != test.wantData[i] {
		t.Errorf("Invalid response, got %q, want %q", got, test.wantData[i])
	}

	for _, challenge := range test.challenges {
		i++
		if i >= len(test.wantData) {
			t.Fatalf("unexpected challenge: %q", challenge)
		}

		toServer, err = auth.Next([]byte(challenge), true)
		if err != nil {
			t.Fatalf("loginAuth.Auth(): %v", err)
		}
		got = string(toServer)
		if got != test.wantData[i] {
			t.Errorf("Invalid response, got %q, want %q", got, test.wantData[i])
		}
	}
}
