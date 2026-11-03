package gemini

import "testing"

func TestNormalizeReceiptMIME(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{name: "defaults to jpeg", input: "", expected: "image/jpeg"},
		{name: "accepts pdf", input: "application/pdf", expected: "application/pdf"},
		{name: "trims and normalizes case", input: "  IMAGE/PNG ", expected: "image/png"},
		{name: "accepts webp", input: "image/webp", expected: "image/webp"},
		{name: "accepts heic", input: "image/heic", expected: "image/heic"},
		{name: "accepts heif", input: "image/heif", expected: "image/heif"},
		{name: "rejects text", input: "text/plain", wantErr: true},
		{name: "rejects empty parameters", input: "application/pdf; charset=utf-8", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeReceiptMIME(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeReceiptMIME() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.expected {
				t.Fatalf("normalizeReceiptMIME() = %q, want %q", got, tt.expected)
			}
		})
	}
}
