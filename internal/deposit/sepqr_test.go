package deposit

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestBuildSEP7URI(t *testing.T) {
	uri := BuildSEP7URI("GABCDEF", "memo-123", "5", "XLM")
	expected := "web+stellar:pay?amount=5&asset_code=XLM&destination=GABCDEF&memo=memo-123&memo_type=text"
	if uri != expected {
		t.Errorf("BuildSEP7URI() = %q, want %q", uri, expected)
	}
}

func TestBuildSEP7URI_URLEncoding(t *testing.T) {
	uri := BuildSEP7URI("GABC DEF", "memo 123", "5.50", "XLM")
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("url.Parse(%q) failed: %v", uri, err)
	}
	q := parsed.Query()
	if q.Get("destination") != "GABC DEF" {
		t.Errorf("destination = %q, want %q", q.Get("destination"), "GABC DEF")
	}
	if q.Get("memo") != "memo 123" {
		t.Errorf("memo = %q, want %q", q.Get("memo"), "memo 123")
	}
}

func TestGenerateQRCode(t *testing.T) {
	uri := "web+stellar:pay?destination=GABCDEF&memo=memo-123&memo_type=text&amount=5&asset_code=XLM"
	dataURI, err := GenerateQRCode(uri)
	if err != nil {
		t.Fatalf("GenerateQRCode() failed: %v", err)
	}

	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Fatalf("GenerateQRCode() prefix = %q, want %q",
			dataURI[:len("data:image/png;base64,")], "data:image/png;base64,")
	}

	b64 := strings.TrimPrefix(dataURI, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	if len(decoded) == 0 {
		t.Fatal("decoded PNG is empty")
	}

	if string(decoded[:4]) != "\x89PNG" {
		t.Fatal("decoded bytes are not a valid PNG")
	}
}
