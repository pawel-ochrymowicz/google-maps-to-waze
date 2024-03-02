package maps

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type LatLng struct {
	Latitude  float64
	Longitude float64
}

type Location interface {
	LatLng() (LatLng, error)
}

const (
	googleMapsLatLngURLRegex     = `(-?\d+\.\d+),\s*(-?\d+\.\d+)`
	googleMapsLatLngContentRegex = `@` + googleMapsLatLngURLRegex
)

var (
	latLngURLPattern     = regexp.MustCompile(googleMapsLatLngURLRegex)
	latLngContentPattern = regexp.MustCompile(googleMapsLatLngContentRegex)
)

type GoogleMapsLink struct {
	latLng LatLng
}

func (l *GoogleMapsLink) LatLng() (LatLng, error) {
	return l.latLng, nil
}

// ParseGoogleMapsFromURL extracts GoogleMapsLink from the given URL.
func ParseGoogleMapsFromURL(u *url.URL, toContent UrlToContent) (*GoogleMapsLink, error) {
	// First, attempt to extract from URL path.
	if latLng, err := latLng(u.Path, latLngURLPattern); err == nil {
		return &GoogleMapsLink{latLng: latLng}, nil
	}

	// If not found in URL path, use the toContent function to get alternative content.
	content, err := toContent(u)
	if err != nil {
		return nil, fmt.Errorf("failed to get content from url: %s, error: %w", u.String(), err)
	}

	// Attempt to extract from the content.
	if latLng, err := latLng(content, latLngContentPattern); err == nil {
		return &GoogleMapsLink{latLng: latLng}, nil
	}

	return nil, fmt.Errorf("failed to find lat lng for url: %s", u.String())
}

func latLng(content string, pattern *regexp.Regexp) (LatLng, error) {
	matches := pattern.FindStringSubmatch(content)
	if matches == nil || len(matches) < 3 {
		return LatLng{}, fmt.Errorf("failed to find latitude and longitude in content")
	}

	lat, err := parsePointFromString(matches[1]) // Assuming matches[2] is latitude based on corrected indices.
	if err != nil {
		return LatLng{}, fmt.Errorf("failed to parse latitude: %w", err)
	}

	lng, err := parsePointFromString(matches[2]) // Assuming matches[1] is longitude based on corrected indices.
	if err != nil {
		return LatLng{}, fmt.Errorf("failed to parse longitude: %w", err)
	}

	// Check if latitude and longitude might be reversed
	if lat < -90 || lat > 90 {
		// Swap values if they are reversed
		lat, lng = lng, lat
	}

	return LatLng{Latitude: lat, Longitude: lng}, nil
}

func parsePointFromString(point string) (float64, error) {
	return strconv.ParseFloat(point, 64)
}

// UrlToContent is a function that takes a URL and returns a string that represents the input to the URL.
type UrlToContent func(u *url.URL) (string, error)

func HttpGetToInput(httpClient *http.Client) func(*url.URL) (string, error) {
	return func(u *url.URL) (string, error) {
		// Create a new HTTP GET request.
		req, err := http.NewRequest("GET", u.String(), http.NoBody)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		// Perform the HTTP GET request.
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to request original URL: %s, error: %w", u.String(), err)
		}
		defer resp.Body.Close()

		// Check the response status code.
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("request to URL: %s returned non-OK status: %d", u.String(), resp.StatusCode)
		}

		// Read the response body.
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}

		return string(bodyBytes), nil
	}
}

type WazeLink struct {
	url *url.URL
}

func (w *WazeLink) URL() *url.URL {
	return w.url
}

const (
	wazeLinkTemplate = "https://www.waze.com/ul?ll=%s&navigate=yes&zoom=5"
)

// WazeFromLocation constructs a new Waze link from location by extracting latitude & longitude
func WazeFromLocation(l Location) (*WazeLink, error) {
	latLng, err := l.LatLng()
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract lat lng from location")
	}

	geoStr := fmt.Sprintf("%.7f,%.7f", latLng.Latitude, latLng.Longitude)
	raw := fmt.Sprintf(wazeLinkTemplate, geoStr)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}
	w := &WazeLink{url: u}
	return w, nil
}
