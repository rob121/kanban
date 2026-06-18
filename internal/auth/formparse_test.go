package auth

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRequestFormURLencoded(t *testing.T) {
	body := "board_id=1&category_id=2&title=test"
	r := httptest.NewRequest("POST", "/cards", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := ParseRequestForm(r); err != nil {
		t.Fatal(err)
	}
	if r.FormValue("board_id") != "1" || r.FormValue("category_id") != "2" {
		t.Fatalf("got board=%q category=%q", r.FormValue("board_id"), r.FormValue("category_id"))
	}
}

func TestParseRequestFormMultipart(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("board_id", "1")
	_ = w.WriteField("category_id", "2")
	_ = w.WriteField("title", "test")
	_ = w.Close()
	r := httptest.NewRequest("POST", "/cards", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	if err := ParseRequestForm(r); err != nil {
		t.Fatal(err)
	}
	if r.FormValue("board_id") != "1" || r.FormValue("category_id") != "2" {
		t.Fatalf("got board=%q category=%q", r.FormValue("board_id"), r.FormValue("category_id"))
	}
}
