package activitylog

import (
	"net/http"
	"time"

	internalutils "github.com/PayRam/activity-log/internal/utils"
	"go.uber.org/zap"
)

// LocationInfo contains geolocation data for an IP address.
type LocationInfo struct {
	Country     string
	CountryCode string
	Region      string
	City        string
	Timezone    string
	Latitude    *float64
	Longitude   *float64
	Success     *bool
	Status      string
	Message     string
	Error       bool
	Reason      string
}

// GeoLookupConfig configures geolocation lookups.
type GeoLookupConfig struct {
	ProviderURLTemplate string
	ProviderName        string
	Timeout             time.Duration
	CacheTTL            time.Duration
	Logger              *zap.Logger
	HTTPClient          *http.Client
}

// GeoLookup resolves public IP addresses to geolocation.
type GeoLookup struct {
	impl *internalutils.GeoLookup
}

var defaultGeoLookup = NewGeoLookup(GeoLookupConfig{})

// NewGeoLookup creates a reusable geolocation lookup instance.
func NewGeoLookup(cfg GeoLookupConfig) *GeoLookup {
	impl := internalutils.NewGeoLookup(internalutils.GeoLookupConfig{
		ProviderURLTemplate: cfg.ProviderURLTemplate,
		ProviderName:        cfg.ProviderName,
		Timeout:             cfg.Timeout,
		CacheTTL:            cfg.CacheTTL,
		Logger:              cfg.Logger,
		HTTPClient:          cfg.HTTPClient,
	})
	return &GeoLookup{impl: impl}
}

// Lookup fetches geolocation for an IP.
func (g *GeoLookup) Lookup(ipAddress string) *LocationInfo {
	if g == nil || g.impl == nil {
		return nil
	}
	return toPublicLocation(g.impl.Lookup(ipAddress))
}

// LookupAsync fetches geolocation asynchronously.
func (g *GeoLookup) LookupAsync(ipAddress string, callback func(*LocationInfo)) {
	if g == nil || g.impl == nil {
		if callback != nil {
			callback(nil)
		}
		return
	}
	g.impl.LookupAsync(ipAddress, func(location *internalutils.LocationInfo) {
		if callback != nil {
			callback(toPublicLocation(location))
		}
	})
}

// ClearCache clears resolver cache.
func (g *GeoLookup) ClearCache() {
	if g == nil || g.impl == nil {
		return
	}
	g.impl.ClearCache()
}

// CacheSize returns cache entry count.
func (g *GeoLookup) CacheSize() int {
	if g == nil || g.impl == nil {
		return 0
	}
	return g.impl.CacheSize()
}

// GetLocationFromIP uses package-level default geolocation lookup.
func GetLocationFromIP(ipAddress string) *LocationInfo {
	return defaultGeoLookup.Lookup(ipAddress)
}

// GetLocationFromIPAsync uses package-level default geolocation lookup asynchronously.
func GetLocationFromIPAsync(ipAddress string, callback func(*LocationInfo)) {
	defaultGeoLookup.LookupAsync(ipAddress, callback)
}

// ClearLocationCache clears package-level geolocation cache.
func ClearLocationCache() {
	defaultGeoLookup.ClearCache()
}

// GetLocationCacheSize returns package-level geolocation cache size.
func GetLocationCacheSize() int {
	return defaultGeoLookup.CacheSize()
}

// EnrichCreateRequestWithLocation maps location values onto a create request.
// Existing request values are preserved.
func EnrichCreateRequestWithLocation(req *CreateRequest, location *LocationInfo) {
	if req == nil || location == nil {
		return
	}

	if req.Country == nil && location.Country != "" {
		req.Country = stringPtr(location.Country)
	}
	if req.CountryCode == nil && location.CountryCode != "" {
		req.CountryCode = stringPtr(location.CountryCode)
	}
	if req.Region == nil && location.Region != "" {
		req.Region = stringPtr(location.Region)
	}
	if req.City == nil && location.City != "" {
		req.City = stringPtr(location.City)
	}
	if req.Timezone == nil && location.Timezone != "" {
		req.Timezone = stringPtr(location.Timezone)
	}
	if req.Latitude == nil && location.Latitude != nil {
		req.Latitude = float64Ptr(*location.Latitude)
	}
	if req.Longitude == nil && location.Longitude != nil {
		req.Longitude = float64Ptr(*location.Longitude)
	}
}

// EnrichUpdateRequestWithLocation maps location values onto an update request.
// Existing request values are preserved.
func EnrichUpdateRequestWithLocation(req *UpdateRequest, location *LocationInfo) {
	if req == nil || location == nil {
		return
	}

	if req.Country == nil && location.Country != "" {
		req.Country = stringPtr(location.Country)
	}
	if req.CountryCode == nil && location.CountryCode != "" {
		req.CountryCode = stringPtr(location.CountryCode)
	}
	if req.Region == nil && location.Region != "" {
		req.Region = stringPtr(location.Region)
	}
	if req.City == nil && location.City != "" {
		req.City = stringPtr(location.City)
	}
	if req.Timezone == nil && location.Timezone != "" {
		req.Timezone = stringPtr(location.Timezone)
	}
	if req.Latitude == nil && location.Latitude != nil {
		req.Latitude = float64Ptr(*location.Latitude)
	}
	if req.Longitude == nil && location.Longitude != nil {
		req.Longitude = float64Ptr(*location.Longitude)
	}
}

func toPublicLocation(location *internalutils.LocationInfo) *LocationInfo {
	if location == nil {
		return nil
	}
	out := &LocationInfo{
		Country:     location.Country,
		CountryCode: location.CountryCode,
		Region:      location.Region,
		City:        location.City,
		Timezone:    location.Timezone,
		Status:      location.Status,
		Message:     location.Message,
		Error:       location.Error,
		Reason:      location.Reason,
	}
	if location.HasLatitude {
		out.Latitude = float64Ptr(location.Latitude)
	}
	if location.HasLongitude {
		out.Longitude = float64Ptr(location.Longitude)
	}
	if location.Success != nil {
		success := *location.Success
		out.Success = &success
	}
	return out
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
