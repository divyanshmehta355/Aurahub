package api

import (
	"net"
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestIsProhibitedIP(t *testing.T) {
	tests := []struct {
		ip         string
		prohibited bool
	}{
		{"127.0.0.1", true},
		{"127.0.1.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"142.250.190.46", false},
	}

	for _, tt := range tests {
		parsed := net.ParseIP(tt.ip)
		if parsed == nil {
			t.Fatalf("failed to parse IP %q", tt.ip)
		}
		got := isProhibitedIP(parsed)
		if got != tt.prohibited {
			t.Errorf("isProhibitedIP(%s) = %v, want %v", tt.ip, got, tt.prohibited)
		}
	}

	if !isProhibitedIP(nil) {
		t.Errorf("isProhibitedIP(nil) must be true (fail closed)")
	}
}

func TestValidatePublicVideoURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/video.mp4", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"localhost host", "http://localhost:8080/video.mp4", true},
		{"internal domain suffix", "https://api.internal/video.mp4", true},
		{"local domain suffix", "https://nas.local/video.mp4", true},
		{"loopback ipv4", "http://127.0.0.1/video.mp4", true},
		{"loopback secondary ipv4", "http://127.0.0.2:8000/video.mp4", true},
		{"loopback ipv6", "http://[::1]/video.mp4", true},
		{"private 10.x", "http://10.0.0.5:8080/video.mp4", true},
		{"private 192.168.x", "http://192.168.1.100/video.mp4", true},
		{"cloud metadata service", "http://169.254.169.254/latest/meta-data/", true},
		{"valid https video URL", "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4", false},
		{"valid http video URL", "http://example.com/stream/bunny.mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePublicVideoURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePublicVideoURL(%q) err = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestIsUserAdmin(t *testing.T) {
	// 1. Regular user with 'admin' in username must NOT be admin
	u1 := db.User{
		Username: "admin_attacker",
		Email:    "attacker@example.com",
		IsAdmin:  pgtype.Bool{Valid: true, Bool: false},
	}
	if isUserAdmin(u1) {
		t.Errorf("user with 'admin' in username was granted admin privileges!")
	}

	// 2. Regular user with 'admin' in email must NOT be admin
	u2 := db.User{
		Username: "normaluser",
		Email:    "admin@otherdomain.com",
		IsAdmin:  pgtype.Bool{Valid: true, Bool: false},
	}
	if isUserAdmin(u2) {
		t.Errorf("user with 'admin' in email was granted admin privileges!")
	}

	// 3. User with IsAdmin == true in DB must be admin
	u3 := db.User{
		Username: "legitadmin",
		Email:    "owner@aurahub.io",
		IsAdmin:  pgtype.Bool{Valid: true, Bool: true},
	}
	if !isUserAdmin(u3) {
		t.Errorf("user with DB IsAdmin=true was not granted admin!")
	}

	// 4. User with IsAdmin invalid (null) must NOT be admin
	u4 := db.User{
		Username: "nulladmin",
		Email:    "nulladmin@aurahub.io",
		IsAdmin:  pgtype.Bool{Valid: false, Bool: false},
	}
	if isUserAdmin(u4) {
		t.Errorf("user with null IsAdmin was granted admin!")
	}
}
