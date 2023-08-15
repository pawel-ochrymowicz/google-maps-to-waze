package maps

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type LatLng struct {
	Latitude  float64
	Longitude float64
}

type Location interface {
	LatLng() (*LatLng, error)
}

const (
	googleMapsLatLngRegex     = `[-]?[\d]+[.][\d]*,[-]?[\d]+[.][\d]*`
	googleMapsLatLngSeparator = ","
)

type GoogleMapsLink struct {
	latLng *LatLng
}

func (l *GoogleMapsLink) LatLng() (*LatLng, error) {
	return l.latLng, nil
}

// ParseGoogleMapsFromURL extracts GoogleMapsLink from the given URL.
func ParseGoogleMapsFromURL(u *url.URL, toInput UrlToInput) (*GoogleMapsLink, error) {
	latLngPattern, err := regexp.Compile(googleMapsLatLngRegex)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize parser")
	}
	latLngLookup := latLng(latLngPattern)
	if latLngPattern.MatchString(u.Path) {
		var latLng *LatLng
		latLng, err = latLngLookup(u.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find lat lng in path: %s", u.Path)
		}
		return &GoogleMapsLink{
			latLng: latLng,
		}, nil
	}
	var input string
	input, err = toInput(u)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get response history from url: %s", u.String())
	}
	if latLngPattern.MatchString(input) {
		var latLng *LatLng
		latLng, err = latLngLookup(input)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find lat lng in response")
		}
		return &GoogleMapsLink{
			latLng: latLng,
		}, nil
	}
	return nil, fmt.Errorf("failed to find lat lng for url: %s", u.String())
}

var errNoLatLng = errors.New("failed to find the lat lng")

func latLng(latLngPattern *regexp.Regexp) func(input string) (*LatLng, error) {
	return func(input string) (*LatLng, error) {
		matches := latLngPattern.FindAllString(input, 10)
		if len(matches) == 0 {
			return nil, errNoLatLng
		}

		latLngString := matches[len(matches)-1]
		latLngParts := strings.Split(latLngString, googleMapsLatLngSeparator)

		if len(latLngParts) != 2 {
			return nil, errors.New("failed to parse lat lng")
		}
		parts := latLngParts
		var lat, lng float64
		lat, err := parsePointFromString(parts[0])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse lat")
		}
		lng, err = parsePointFromString(parts[1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse lng")
		}

		latLng := &LatLng{Latitude: lat, Longitude: lng}
		return latLng, nil
	}
}

func parsePointFromString(point string) (float64, error) {
	return strconv.ParseFloat(point, 64)
}

// UrlToInput is a function that takes a URL and returns a string that represents the input to the URL.
type UrlToInput func(u *url.URL) (string, error)

func HttpGetToInput(httpClient *http.Client) func(*url.URL) (string, error) {
	return func(u *url.URL) (string, error) {
		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", errors.Wrapf(err, "failed to request original url: %s", u.String())
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "failed to read response body")
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
