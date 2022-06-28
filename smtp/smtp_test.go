package smtp

import (
	"bytes"
	"crypto/tls"
	"github.com/hacku7/gomail/msg"
	"io"
	"net"
	"net/smtp"
	"reflect"
	"testing"
	"time"
)

const (
	testPort    = 587
	testSSLPort = 465
	TestUser    = "user"
	TestPwd     = "pwd"
	TestHost    = "smtp.example.com"
	testTo1     = "to1@example.com"
	testTo2     = "to2@example.com"
	testFrom    = "from@example.com"
	testBody    = "Test msg"
	testMsg     = "To: " + testTo1 + ", " + testTo2 + "\r\n" +
		"From: " + testFrom + "\r\n" +
		"Mime-Version: 1.0\r\n" +
		"Date: Wed, 25 Jun 2014 17:46:00 +0000\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		testBody
)

var (
	testConn    = &net.TCPConn{}
	testTLSConn = &tls.Conn{}
	testConfig  = &tls.Config{InsecureSkipVerify: true}
	testAuth    = smtp.PlainAuth("", TestUser, TestPwd, TestHost)
)

func TestDialer(t *testing.T) {
	d := NewDialer(TestHost, testPort, "user", "pwd")
	testSendMail(t, d, []string{
		"Extension STARTTLS",
		"StartTLS",
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestDialerSSL(t *testing.T) {
	d := NewDialer(TestHost, testSSLPort, "user", "pwd")
	testSendMail(t, d, []string{
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestDialerConfig(t *testing.T) {
	d := NewDialer(TestHost, testPort, "user", "pwd")
	d.LocalName = "test"
	d.TLSConfig = testConfig
	testSendMail(t, d, []string{
		"Hello test",
		"Extension STARTTLS",
		"StartTLS",
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestDialerSSLConfig(t *testing.T) {
	d := NewDialer(TestHost, testSSLPort, "user", "pwd")
	d.LocalName = "test"
	d.TLSConfig = testConfig
	testSendMail(t, d, []string{
		"Hello test",
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestDialerNoAuth(t *testing.T) {
	d := &Dialer{
		Host: TestHost,
		Port: testPort,
	}
	testSendMail(t, d, []string{
		"Extension STARTTLS",
		"StartTLS",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestDialerTimeout(t *testing.T) {
	d := &Dialer{
		Host: TestHost,
		Port: testPort,
	}
	testSendMailTimeout(t, d, []string{
		"Extension STARTTLS",
		"StartTLS",
		"Mail " + testFrom,
		"Extension STARTTLS",
		"StartTLS",
		"Mail " + testFrom,
		"Rcpt " + testTo1,
		"Rcpt " + testTo2,
		"Data",
		"Write msg",
		"Close writer",
		"Quit",
		"Close",
	})
}

type mockClient struct {
	t       *testing.T
	i       int
	want    []string
	addr    string
	config  *tls.Config
	timeout bool
}

func (c *mockClient) Hello(localName string) error {
	c.do("Hello " + localName)
	return nil
}

func (c *mockClient) Extension(ext string) (bool, string) {
	c.do("Extension " + ext)
	return true, ""
}

func (c *mockClient) StartTLS(config *tls.Config) error {
	assertConfig(c.t, config, c.config)
	c.do("StartTLS")
	return nil
}

func (c *mockClient) Auth(a smtp.Auth) error {
	if !reflect.DeepEqual(a, testAuth) {
		c.t.Errorf("Invalid auth, got %#v, want %#v", a, testAuth)
	}
	c.do("Auth")
	return nil
}

func (c *mockClient) Mail(from string) error {
	c.do("Mail " + from)
	if c.timeout {
		c.timeout = false
		return io.EOF
	}
	return nil
}

func (c *mockClient) Rcpt(to string) error {
	c.do("Rcpt " + to)
	return nil
}

func (c *mockClient) Data() (io.WriteCloser, error) {
	c.do("Data")
	return &mockWriter{c: c, want: testMsg}, nil
}

func (c *mockClient) Quit() error {
	c.do("Quit")
	return nil
}

func (c *mockClient) Close() error {
	c.do("Close")
	return nil
}

func (c *mockClient) do(cmd string) {
	if c.i >= len(c.want) {
		c.t.Fatalf("Invalid command %q", cmd)
	}

	if cmd != c.want[c.i] {
		c.t.Fatalf("Invalid command, got %q, want %q", cmd, c.want[c.i])
	}
	c.i++
}

type mockWriter struct {
	want string
	c    *mockClient
	buf  bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (int, error) {
	if w.buf.Len() == 0 {
		w.c.do("Write msg")
	}
	w.buf.Write(p)
	return len(p), nil
}

func (w *mockWriter) Close() error {
	msg.CompareBodies(w.c.t, w.buf.String(), w.want)
	w.c.do("Close writer")
	return nil
}

func testSendMail(t *testing.T, d *Dialer, want []string) {
	doTestSendMail(t, d, want, false)
}

func testSendMailTimeout(t *testing.T, d *Dialer, want []string) {
	doTestSendMail(t, d, want, true)
}

func doTestSendMail(t *testing.T, d *Dialer, want []string, timeout bool) {
	testClient := &mockClient{
		t:       t,
		want:    want,
		addr:    addr(d.Host, d.Port),
		config:  d.TLSConfig,
		timeout: timeout,
	}

	netDialTimeout = func(network, address string, d time.Duration) (net.Conn, error) {
		if network != "tcp" {
			t.Errorf("Invalid network, got %q, want tcp", network)
		}
		if address != testClient.addr {
			t.Errorf("Invalid address, got %q, want %q",
				address, testClient.addr)
		}
		return testConn, nil
	}

	tlsClient = func(conn net.Conn, config *tls.Config) *tls.Conn {
		if conn != testConn {
			t.Errorf("Invalid conn, got %#v, want %#v", conn, testConn)
		}
		assertConfig(t, config, testClient.config)
		return testTLSConn
	}

	smtpNewClient = func(conn net.Conn, host string) (smtpClient, error) {
		if host != TestHost {
			t.Errorf("Invalid host, got %q, want %q", host, TestHost)
		}
		return testClient, nil
	}

	if err := d.DialAndSend(getTestMessage()); err != nil {
		t.Error(err)
	}
}

func getTestMessage() *msg.Message {
	m := msg.NewMessage()
	m.SetHeader("From", testFrom)
	m.SetHeader("To", testTo1, testTo2)
	m.SetBody("text/plain", testBody)

	return m
}

func assertConfig(t *testing.T, got, want *tls.Config) {
	if want == nil {
		want = &tls.Config{ServerName: TestHost}
	}
	if got.ServerName != want.ServerName {
		t.Errorf("Invalid field ServerName in config, got %q, want %q", got.ServerName, want.ServerName)
	}
	if got.InsecureSkipVerify != want.InsecureSkipVerify {
		t.Errorf("Invalid field InsecureSkipVerify in config, got %v, want %v", got.InsecureSkipVerify, want.InsecureSkipVerify)
	}
}
