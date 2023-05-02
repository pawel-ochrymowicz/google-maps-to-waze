package maps

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type LatLng struct {
	Latitude  string
	Longitude string
}

type Location interface {
	LatLng() (*LatLng, error)
}

const (
	googleMapsLatLngRegex     = `([0-9]+\.[0-9]{7},[0-9]+\.[0-9]{7})`
	googleMapsLatLngSeparator = ","
)

type GoogleMapsLink struct {
	latLng *LatLng
}

func (l *GoogleMapsLink) LatLng() (*LatLng, error) {
	return l.latLng, nil
}

type ToInput func(u *url.URL) (string, error)

func HttpGetToInput(httpClient *http.Client) func(*url.URL) (string, error) {
	return func(u *url.URL) (string, error) {
		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", errors.Wrap(err, "failed to request original url")
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "failed to read response body")
		}
		return string(bodyBytes), nil
	}
}

// ParseGoogleMapsFromURL extracts GoogleMapsLink from the given URL.
func ParseGoogleMapsFromURL(u *url.URL, toInput ToInput) (*GoogleMapsLink, error) {
	latLngPattern, err := regexp.Compile(googleMapsLatLngRegex)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize parser")
	}
	input := u.Path

	if !latLngPattern.MatchString(input) {
		input, err = toInput(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve shortened URL")
		}
	}

	matches := latLngPattern.FindAllString(input, 10)
	if len(matches) == 0 {
		return nil, fmt.Errorf("failed to find the lat lng: %s", input)
	}

	latLngString := matches[len(matches)-1]
	latLngParts := strings.Split(latLngString, googleMapsLatLngSeparator)

	if len(latLngParts) != 2 {
		return nil, errors.New("failed to parse lat lng")
	}

	latLng := &LatLng{Latitude: latLngParts[0], Longitude: latLngParts[1]}

	return &GoogleMapsLink{
		latLng: latLng,
	}, nil
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

	geoStr := fmt.Sprintf("%s,%s", latLng.Latitude, latLng.Longitude)
	raw := fmt.Sprintf(wazeLinkTemplate, geoStr)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}
	w := &WazeLink{url: u}
	return w, nil
}

// GoogleMapsUrlToWazeLink is just a facade to convert from Google Maps to Waze
func GoogleMapsUrlToWazeLink(u *url.URL, toInput ToInput) (*WazeLink, error) {
	g, err := ParseGoogleMapsFromURL(u, toInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Google Maps from URL")
	}
	return WazeFromLocation(g)
}
