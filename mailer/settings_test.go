package mailer

import "testing"

func TestConfigure(t *testing.T) {
	Configure(Settings{
		Host:     "smtp.example.com",
		Port:     0,
		User:     "mailer",
		Password: "secret",
		From:     "noreply@example.com",
		FromName: "Kanban",
		TestTo:   "dev@example.com",
		BaseURL:  "https://kanban.example.com/",
		SiteName: "My Boards",
		BrandMark: "TB",
		BrandColor: "#4f46e5",
	})

	if Host != "smtp.example.com" {
		t.Fatalf("Host = %q", Host)
	}
	if Port != 587 {
		t.Fatalf("Port = %d, want default 587", Port)
	}
	if !TestMode || TestTo != "dev@example.com" {
		t.Fatalf("test mode not enabled correctly: %v %q", TestMode, TestTo)
	}
	if defaultMailerBaseURL() != "https://kanban.example.com" {
		t.Fatalf("BaseURL = %q", defaultMailerBaseURL())
	}
	if defaultMailerSiteName() != "My Boards" {
		t.Fatalf("SiteName = %q", defaultMailerSiteName())
	}
	if defaultMailerBrandMark() != "TB" {
		t.Fatalf("BrandMark = %q", defaultMailerBrandMark())
	}
	if defaultMailerBrandColor() != "#4f46e5" {
		t.Fatalf("BrandColor = %q", defaultMailerBrandColor())
	}
	if !Enabled() {
		t.Fatal("expected Enabled() true when host is set")
	}
}

func TestEnabledWithoutHost(t *testing.T) {
	Configure(Settings{})
	if Enabled() {
		t.Fatal("expected Enabled() false when host is empty")
	}
}
