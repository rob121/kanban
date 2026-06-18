// Package mailer sends HTML email over SMTP (gomail) using package-level configuration.
//
// Call Configure from application startup after loading config.json (mail.host, mail.port, etc.).
package mailer

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	mail "gopkg.in/gomail.v2"
)

var TmplPath string
var Port int
var UserName string
var Password string
var Host string
var From string

// FromName is the optional display name for the From header ("Name <address>"); empty uses address only.
var FromName string

// ReplyTo is an optional Reply-To address; empty omits the header.
var ReplyTo string
var TestMode bool
var TestTo string

// Logo is an optional inline logo path for HTML emails.
var Logo string

// RefreshLogoFromSettings is a no-op hook for future branding overrides.
func RefreshLogoFromSettings() {}

// ResolveEmailContent loads email body HTML for a named template class.
var ResolveEmailContent func(telegramClass string, args map[string]interface{}) (template.HTML, error)

// SendCallback, if non-nil, is invoked after each Send() completes (success or failure) with a snapshot
// of headers and rendered body for auditing. Callbacks must not block unreasonably; panics are recovered.
var SendCallback func(SendLogSnapshot)

// SendLogSnapshot is a value copy of send metadata plus rendered HTML body (truncated).
type SendLogSnapshot struct {
	SentAt    time.Time
	From      string
	To        string
	Cc        string
	Bcc       string
	Subject   string
	BodyHTML  string
	BodyBytes int // original rendered size before truncation
	Success   bool
	Error     string
	TestMode  bool
}

const maxSendLogBodyBytes = 512 * 1024

func mergeDefaultEmailArgs(into map[string]interface{}) {
	if into == nil {
		return
	}
	if _, ok := into["BaseURL"]; !ok {
		into["BaseURL"] = defaultMailerBaseURL()
	}
	if _, ok := into["SiteName"]; !ok {
		into["SiteName"] = defaultMailerSiteName()
	}
	if _, ok := into["BrandMark"]; !ok {
		into["BrandMark"] = defaultMailerBrandMark()
	}
	if _, ok := into["BrandColor"]; !ok {
		into["BrandColor"] = defaultMailerBrandColor()
	}
}

// MergeDefaultEmailArgs adds BaseURL, SiteName, and branding when keys are absent.
// Call before ResolveEmailContent when bypassing (*Msg).Content.
func MergeDefaultEmailArgs(into map[string]interface{}) {
	mergeDefaultEmailArgs(into)
}

func mergeDefaultEmailArgsAny(into map[string]any) {
	if into == nil {
		return
	}
	if _, ok := into["BaseURL"]; !ok {
		into["BaseURL"] = defaultMailerBaseURL()
	}
	if _, ok := into["SiteName"]; !ok {
		into["SiteName"] = defaultMailerSiteName()
	}
	if _, ok := into["BrandMark"]; !ok {
		into["BrandMark"] = defaultMailerBrandMark()
	}
	if _, ok := into["BrandColor"]; !ok {
		into["BrandColor"] = defaultMailerBrandColor()
	}
}

func joinMailHeader(m *mail.Message, field string) string {
	if m == nil {
		return ""
	}
	return strings.Join(m.GetHeader(field), ", ")
}

func newSendLogSnapshot(msg *Msg, bodyHTML string, sendErr error) SendLogSnapshot {
	snap := SendLogSnapshot{
		SentAt:    time.Now().UTC(),
		From:      joinMailHeader(msg.Message, "From"),
		To:        joinMailHeader(msg.Message, "To"),
		Cc:        joinMailHeader(msg.Message, "Cc"),
		Bcc:       joinMailHeader(msg.Message, "Bcc"),
		Subject:   joinMailHeader(msg.Message, "Subject"),
		BodyBytes: len(bodyHTML),
		Success:   sendErr == nil,
		TestMode:  TestMode,
	}
	if sendErr != nil {
		snap.Error = sendErr.Error()
	}
	if len(bodyHTML) > maxSendLogBodyBytes {
		snap.BodyHTML = bodyHTML[:maxSendLogBodyBytes]
	} else {
		snap.BodyHTML = bodyHTML
	}
	return snap
}

// RecordSendResult writes one mail send audit row via SendCallback when configured.
func RecordSendResult(msg *Msg, bodyHTML string, sendErr error) {
	if SendCallback == nil || msg == nil {
		return
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("mailer RecordSendResult panic: %v", r)
			}
		}()
		SendCallback(newSendLogSnapshot(msg, bodyHTML, sendErr))
	}()
}

type Msg struct {
	Message *mail.Message
	Args    map[string]interface{}
	Errors  []error
}

func New() *Msg {
	msg := &Msg{}
	msg.Message = mail.NewMessage()
	msg.Args = make(map[string]interface{})
	mergeDefaultEmailArgs(msg.Args)
	fromAddr := strings.TrimSpace(From)
	if fromAddr == "" {
		fromAddr = strings.TrimSpace(UserName)
	}
	fn := strings.TrimSpace(FromName)
	if fn != "" && fromAddr != "" {
		msg.Message.SetAddressHeader("From", fromAddr, fn)
	} else if fromAddr != "" {
		msg.Message.SetHeader("From", fromAddr)
	}
	if rt := strings.TrimSpace(ReplyTo); rt != "" {
		msg.Message.SetHeader("Reply-To", rt)
	}
	return msg
}

func (msg *Msg) To(to ...string) *Msg {
	if TestMode {
		msg.Message.SetHeader("To", TestTo)
	} else {
		msg.Message.SetHeader("To", to...)
	}
	return msg
}

func (msg *Msg) Cc(to ...string) *Msg {
	if TestMode {
		return msg
	}
	msg.Message.SetHeader("Cc", to...)
	return msg
}

func (msg *Msg) Bcc(to ...string) *Msg {
	if TestMode {
		return msg
	}
	msg.Message.SetHeader("Bcc", to...)
	return msg
}

func (msg *Msg) From(from string) *Msg {
	msg.Message.SetHeader("From", from)
	return msg
}

func (msg *Msg) Subject(s string, argsRaw ...map[string]any) *Msg {
	var args map[string]any
	if len(argsRaw) > 0 {
		args = argsRaw[0]
	}
	if args == nil {
		args = map[string]any{}
	}
	mergeDefaultEmailArgsAny(args)
	tm, tmerr := template.New("subject").Funcs(templateFuncs()).Parse(s)
	if tmerr != nil {
		msg.Errors = append(msg.Errors, tmerr)
	}
	var b bytes.Buffer
	err := tm.Execute(&b, args)
	if err != nil {
		msg.Errors = append(msg.Errors, err)
	}
	msg.Message.SetHeader("Subject", b.String())
	return msg
}

func (msg *Msg) Body(s template.HTML) *Msg {
	msg.Add("Body", s)
	return msg
}

func (msg *Msg) Add(key string, val interface{}) *Msg {
	msg.Args[key] = val
	return msg
}

// Embed attaches filename as an inline MIME part (for HTML img src). It returns template.URL("cid:<basename>")
// matching gomail's Embed convention.
func (msg *Msg) Embed(filename string) template.URL {
	path := strings.TrimSpace(filename)
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	msg.Message.Embed(path)
	base := filepath.Base(path)
	if base == "" || base == "." {
		return ""
	}
	return template.URL("cid:" + base)
}

func (msg *Msg) Content(telegramClass string, args map[string]interface{}) *Msg {
	if ResolveEmailContent == nil {
		msg.Errors = append(msg.Errors, errors.New("mailer.ResolveEmailContent callback is not configured"))
		return msg
	}
	if args == nil {
		args = make(map[string]interface{})
	}
	mergeDefaultEmailArgs(args)
	htmlResult, err := ResolveEmailContent(telegramClass, args)
	if err != nil {
		msg.Errors = append(msg.Errors, err)
		if htmlResult != "" {
			msg.Body(htmlResult)
		}
		return msg
	}
	for k, v := range args {
		msg.Add(k, v)
	}
	msg.Body(htmlResult)
	return msg
}

// SendTestMail sends one diagnostic HTML message to `to` using the current package-level SMTP settings.
// The recipient is always `to`, never redirected by TestMode.
func SendTestMail(to string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("recipient address is required")
	}
	if _, err := netmail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	host := strings.TrimSpace(Host)
	if host == "" {
		return errors.New("SMTP host is not configured")
	}
	fromAddr := strings.TrimSpace(From)
	if fromAddr == "" {
		fromAddr = strings.TrimSpace(UserName)
	}
	if fromAddr == "" {
		return errors.New("from address is empty; set mail.from or mail.user")
	}

	m := mail.NewMessage()
	fn := strings.TrimSpace(FromName)
	if fn != "" {
		m.SetAddressHeader("From", fromAddr, fn)
	} else {
		m.SetHeader("From", fromAddr)
	}
	if rt := strings.TrimSpace(ReplyTo); rt != "" {
		m.SetHeader("Reply-To", rt)
	}
	m.SetHeader("To", to)
	site := strings.TrimSpace(defaultMailerSiteName())
	subj := "SMTP test"
	if site != "" {
		subj = site + " — SMTP test"
	}
	m.SetHeader("Subject", subj)

	inner := `<p>This is a test message from <strong>` + template.HTMLEscapeString(site) + `</strong>.</p><p>If you are reading this, outbound SMTP is working.</p>`
	args := map[string]interface{}{
		"Body": template.HTML(inner),
	}
	mergeDefaultEmailArgs(args)

	tmplPath := strings.TrimSpace(TmplPath)
	body, err := renderTemplate(tmplPath, args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		body = "<!DOCTYPE html><html><head><meta charset=\"utf-8\"></head><body>" + inner + "</body></html>"
	}
	m.SetBody("text/html", body)

	d := mail.NewDialer(Host, Port, UserName, Password)
	if Port == 465 {
		d.SSL = true
	}
	return d.DialAndSend(m)
}

func (msg *Msg) sendHTMLBody(body string) error {
	d := mail.NewDialer(Host, Port, UserName, Password)
	if Port == 465 {
		d.SSL = true
	}
	msg.Message.SetBody("text/html", body)
	if TestMode {
		log.Printf("mailer: test mode send to %s: %+v", TestTo, msg.Message)
	}
	err2 := d.DialAndSend(msg.Message)
	if err2 != nil {
		msg.Errors = append(msg.Errors, err2)
	}
	if SendCallback != nil {
		snap := newSendLogSnapshot(msg, body, err2)
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("mailer SendCallback panic: %v", r)
				}
			}()
			SendCallback(snap)
		}()
	}
	if TestMode && len(msg.Errors) > 0 {
		log.Printf("mailer: send errors: %+v", msg.Errors)
	}
	return err2
}

func (msg *Msg) Send() error {
	if msg.Args == nil {
		msg.Args = make(map[string]interface{})
	}

	body, err := renderTemplate(TmplPath, msg.Args)
	if err != nil {
		msg.Errors = append(msg.Errors, err)
		return err
	}
	if strings.TrimSpace(body) == "" {
		if h, ok := msg.Args["Body"].(template.HTML); ok && strings.TrimSpace(string(h)) != "" {
			inner := string(h)
			body = "<!DOCTYPE html><html><head><meta charset=\"utf-8\"></head><body>" + inner + "</body></html>"
		}
	}

	return msg.sendHTMLBody(body)
}
