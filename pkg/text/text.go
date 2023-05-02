package text

import (
	"github.com/pkg/errors"
	"net/url"
	"regexp"
)

const (
	urlRegex = "(http|ftp|https):\\/\\/([\\w_-]+(?:(?:\\.[\\w_-]+)+))([\\w.,@?^=%&:\\/~+#-]*[\\w@?^=%&\\/~+#-])"
)

// ParseFirstUrl attempts to parse the first URL found in the given text using a regular expression.
// If a valid URL is found, a pointer to a `url.URL` struct containing information about the parsed URL is returned.
// If no URL is found, an error is returned.
func ParseFirstUrl(text string) (*url.URL, error) {
	r, err := regexp.Compile(urlRegex)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize url parser")
	}
	matched := r.FindString(text)
	return url.Parse(matched)
}
