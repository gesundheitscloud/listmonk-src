package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/smtppool"
)

const (
	emName        = "email"
	hdrReturnPath = "Return-Path"
)

// define error messages
var (
	ErrEmailNoSMTPServerForFrom = errors.New("no applicable SMTP server for FROM address")
	ErrEmailInvalidFromAddress  = errors.New("invalid FROM email address")
)

// Server represents an SMTP server's credentials.
type Server struct {
	Username             string            `json:"username"`
	Password             string            `json:"password"`
	AuthProtocol         string            `json:"auth_protocol"`
	TLSType              string            `json:"tls_type"`
	TLSSkipVerify        bool              `json:"tls_skip_verify"`
	EmailHeaders         map[string]string `json:"email_headers"`
	AllowedFromAddresses []string          `json:"allowed_from_addresses"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

func (s *Server) allowsFromAddress(emailAddress string) bool {
	// leave empty to allow all
	if len(s.AllowedFromAddresses) == 0 {
		return true
	}

	domain := strings.Split(emailAddress, "@")[1]
	for _, allowed := range s.AllowedFromAddresses {
		if emailAddress == allowed || domain == allowed {
			return true
		}
	}
	return false
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	servers []*Server
}

// New returns an SMTP e-mail Messenger backend with the given SMTP servers.
func New(servers ...Server) (*Emailer, error) {
	e := &Emailer{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		var auth smtp.Auth
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
		case "", "none":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}
		s.Opt.Auth = auth

		// TLS config.
		if s.TLSType != "none" {
			s.TLSConfig = &tls.Config{}
			if s.TLSSkipVerify {
				s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
			} else {
				s.TLSConfig.ServerName = s.Host
			}

			// SSL/TLS, not STARTTLS.
			if s.TLSType == "TLS" {
				s.Opt.SSL = true
			}
		}

		pool, err := smtppool.New(s.Opt)
		if err != nil {
			return nil, err
		}

		s.pool = pool
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

// Name returns the Server's name.
func (e *Emailer) Name() string {
	return emName
}

func applicableServers(servers []*Server, fromAddress string) (applServers []*Server, err error) {
	parsedFromAddress, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return applServers, ErrEmailInvalidFromAddress
	}

	for _, smtpServer := range servers {
		if smtpServer.allowsFromAddress(parsedFromAddress.Address) {
			applServers = append(applServers, smtpServer)
		}
	}
	return
}

// Push pushes a message to the server.
func (e *Emailer) Push(m messenger.Message) error {
	applicableServers, err := applicableServers(e.servers, m.From)
	if err != nil {
		return err
	}

	// If there are more than one SMTP servers, send to a random
	// one from the list.
	var (
		ln  = len(applicableServers)
		srv *Server
	)

	if ln > 0 {
		srv = applicableServers[rand.Intn(ln)]
	} else {
		return ErrEmailNoSMTPServerForFrom
	}

	// Are there attachments?
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	em := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Subject:     m.Subject,
		Attachments: files,
	}

	em.Headers = textproto.MIMEHeader{}

	// Attach SMTP level headers.
	for k, v := range srv.EmailHeaders {
		em.Headers.Set(k, v)
	}

	// Attach e-mail level headers.
	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	// If the `Return-Path` header is set, it should be set as the
	// the SMTP envelope sender (via the Sender field of the email struct).
	if sender := em.Headers.Get(hdrReturnPath); sender != "" {
		em.Sender = sender
		em.Headers.Del(hdrReturnPath)
	}

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Body)
	default:
		em.HTML = m.Body
		if len(m.AltBody) > 0 {
			em.Text = m.AltBody
		}
	}

	return srv.pool.Send(em)
}

// Flush flushes the message queue to the server.
func (e *Emailer) Flush() error {
	return nil
}

// Close closes the SMTP pools.
func (e *Emailer) Close() error {
	for _, s := range e.servers {
		s.pool.Close()
	}
	return nil
}
