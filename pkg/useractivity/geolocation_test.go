package useractivity

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGeoLookupLookupAndCache(t *testing.T) {
	var calls int32

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return jsonResponse(`{
			"country":"United States",
			"country_code":"US",
			"region":"California",
			"city":"Mountain View",
			"timezone":"America/Los_Angeles",
			"latitude":37.3861,
			"longitude":-122.0839
		}`), nil
		}),
	}

	lookup := NewGeoLookup(GeoLookupConfig{
		ProviderURLTemplate: "https://geo.local/json/%s",
		CacheTTL:            time.Hour,
		HTTPClient:          client,
	})

	first := lookup.Lookup("8.8.8.8")
	if first == nil {
		t.Fatalf("expected location")
	}
	if first.Country != "United States" || first.City != "Mountain View" {
		t.Fatalf("unexpected location payload: %+v", first)
	}

	second := lookup.Lookup("8.8.8.8")
	if second == nil {
		t.Fatalf("expected cached location")
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected one provider call, got %d", got)
	}
}

func TestGeoLookupSkipsPrivateIP(t *testing.T) {
	var calls int32
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return jsonResponse(`{"country":"x"}`), nil
		}),
	}

	lookup := NewGeoLookup(GeoLookupConfig{
		ProviderURLTemplate: "https://geo.local/json/%s",
		HTTPClient:          client,
	})

	if location := lookup.Lookup("10.1.2.3"); location != nil {
		t.Fatalf("expected nil location for private ip")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected provider not to be called, got %d calls", got)
	}
}

func TestGeoLookupFailurePayload(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(`{"success":false,"message":"reserved range"}`), nil
		}),
	}

	lookup := NewGeoLookup(GeoLookupConfig{
		ProviderURLTemplate: "https://geo.local/json/%s",
		HTTPClient:          client,
	})

	if location := lookup.Lookup("8.8.4.4"); location != nil {
		t.Fatalf("expected nil location for provider failure payload")
	}
}

func TestEnrichRequestWithLocation(t *testing.T) {
	location := &LocationInfo{
		Country:     "United States",
		CountryCode: "US",
		Region:      "California",
		City:        "Mountain View",
		Timezone:    "America/Los_Angeles",
		Latitude:    37.3861,
		Longitude:   -122.0839,
	}

	createReq := &CreateRequest{}
	EnrichCreateRequestWithLocation(createReq, location)
	if createReq.Country == nil || *createReq.Country != "United States" {
		t.Fatalf("expected create request country")
	}
	if createReq.Latitude == nil || *createReq.Latitude != 37.3861 {
		t.Fatalf("expected create request latitude")
	}

	updateReq := &UpdateRequest{}
	EnrichUpdateRequestWithLocation(updateReq, location)
	if updateReq.City == nil || *updateReq.City != "Mountain View" {
		t.Fatalf("expected update request city")
	}
	if updateReq.Longitude == nil || *updateReq.Longitude != -122.0839 {
		t.Fatalf("expected update request longitude")
	}
}
