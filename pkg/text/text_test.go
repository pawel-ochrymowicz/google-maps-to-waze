package text

import (
	"net/url"
	"testing"
)

func TestParseFirstUrl_ValidURL(t *testing.T) {
	text := "Visit our website at https://www.example.com"
	expectedURL, _ := url.Parse("https://www.example.com")
	expectedError := error(nil)

	actualURL, actualError := ParseFirstUrl(text)

	if actualError != expectedError {
		t.Errorf("Expected no error but got %v", actualError)
	}

	if actualURL.String() != expectedURL.String() {
		t.Errorf("Expected URL %q but got %q", expectedURL, actualURL)
	}
}
