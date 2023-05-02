package maps

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGoogleMapsFromURL(t *testing.T) {
	testCases := []struct {
		name          string
		inputURL      string
		expectedError string
		expectedLink  *GoogleMapsLink
	}{
		{
			name:     "Valid URL with lat-lng",
			inputURL: "https://www.google.com/maps/place/37.4219999,122.0840575",
			expectedLink: &GoogleMapsLink{
				latLng: &LatLng{
					Latitude:  "37.4219999",
					Longitude: "122.0840575",
				},
			},
		},
		{
			name:          "Invalid URL",
			inputURL:      "https://www.example.com",
			expectedError: "failed to find the lat lng: ",
		},
	}

	httpClient := &http.Client{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputURL, err := url.Parse(tc.inputURL)
			require.NoError(t, err)

			link, err := ParseGoogleMapsFromURL(inputURL, HttpGetToInput(httpClient))

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLink.latLng, link.latLng)
			}
		})
	}
}

func TestParseGoogleMapsFromURL_Shortened(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	// Set up a test server to handle the shortened URL request.
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/12345" {
			fmt.Printf("%s", r.URL)
			w.Header().Set("Location", fmt.Sprintf("http://localhost:%d/maps/place/37.4219999,122.0840575", port))
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/maps/place/37.4219999,122.0840575" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("AAACAAAAAAAAAAAAAAAAAAAAAA\\\",null,null,[[[1,6]],0,null,2,4]],null,\\\"06-500 Cegielnia Lewicka\\\",null,null,\\\"https://www.google.com/maps/preview/place/Paintball+M%C5%82awa+-+Paintball+Laserowy+LaserTag,+06-500+Cegielnia+Lewicka/@53.1344674,20.3160387,2394a,13.1y/data\\\\u003d!4m2!3m1!1s0x471db6975feccd7d:0xf56f56c92a0d0c73\\\",1,null,null,null,null,null,null,null,null,[[[[\\\"https://www.google.com/maps/contrib/113380515676477458146?hl\\\\u003dru\\\",\\\"Agata Świątek\\\",\\\"https://lh3.googleusercontent.com/a-/ACB-R5TKahdtMYMU9LBX0ziGXdQlwZhRQuFzfTpjMga4_g\\\\u003ds120-c-c0x00000000-cc-rp-mo-br100\\\",\\\"0ahUKEwi8x6naiNX-AhWPqIsKHWk4AisQ4h4IAygA\\\",\\\",AOvVaw3aCQ10FxiG7JEQAKG5Qbqg,,0ahUKEwjY9KfaiNX-AhWPqIsKHWk4AisQ4h4IKygA,\\\"],\\\"5 месяцев назад\\\",null,\\\"Firma n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	testServer.Listener = listener
	testServer.Start()
	defer testServer.Close()
	u, err := url.Parse(testServer.URL + "/12345")
	require.NoError(t, err)

	httpClient := &http.Client{}
	link, err := ParseGoogleMapsFromURL(u, HttpGetToInput(httpClient))

	require.NoError(t, err)
	assert.Equal(t, link, &GoogleMapsLink{
		latLng: &LatLng{
			Latitude:  "53.1344674",
			Longitude: "20.3160387",
		},
	})
}
