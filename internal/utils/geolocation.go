package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultGeoProviderURLTemplate = "https://ipwhois.app/json/%s"
	defaultGeoProviderName        = "ipwhois.io"
	defaultGeoTimeout             = 5 * time.Second
	defaultGeoCacheTTL            = 24 * time.Hour

	geoProviderURLEnv  = "GEOLOCATION_PROVIDER_URL"
	geoProviderNameEnv = "GEOLOCATION_PROVIDER_NAME"
)

// LocationInfo holds geolocation data for an IP address.
// Supports multiple providers: ipwhois.io, ipapi.co, ipwho.is, ip-api.com.
type LocationInfo struct {
	Country      string  `json:"country"`
	CountryCode  string  `json:"country_code"`
	Region       string  `json:"region"`
	City         string  `json:"city"`
	Timezone     string  `json:"timezone"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	HasLatitude  bool    `json:"-"`
	HasLongitude bool    `json:"-"`
	Success      *bool   `json:"success"` // For ipwhois style responses.
	Status       string  `json:"status"`  // For ip-api compatibility.
	Message      string  `json:"message"` // For provider error payloads.
	Error        bool    `json:"error"`   // For ipapi error payloads.
	Reason       string  `json:"reason"`  // For ipapi error payloads.
}

// UnmarshalJSON normalizes payloads from multiple geolocation providers.
func (l *LocationInfo) UnmarshalJSON(data []byte) error {
	type alias struct {
		Country      string  `json:"country"`
		CountryName  string  `json:"country_name"`
		CountryCode  string  `json:"country_code"`
		CountryCode2 string  `json:"countryCode"`
		Region       string  `json:"region"`
		RegionName   string  `json:"regionName"`
		City         string  `json:"city"`
		Lat          float64 `json:"lat"`
		Latitude     float64 `json:"latitude"`
		Lon          float64 `json:"lon"`
		Longitude    float64 `json:"longitude"`
		Success      bool    `json:"success"`
		Status       string  `json:"status"`
		Message      string  `json:"message"`
		Error        bool    `json:"error"`
		Reason       string  `json:"reason"`
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.CountryName != "" {
		l.Country = aux.CountryName
	} else {
		l.Country = aux.Country
	}

	if aux.CountryCode != "" {
		l.CountryCode = aux.CountryCode
	} else {
		l.CountryCode = aux.CountryCode2
	}

	if aux.Region != "" {
		l.Region = aux.Region
	} else {
		l.Region = aux.RegionName
	}

	l.City = aux.City

	if tzRaw, ok := raw["timezone"]; ok {
		switch tz := tzRaw.(type) {
		case string:
			l.Timezone = tz
		case map[string]interface{}:
			if id, ok := tz["id"].(string); ok {
				l.Timezone = id
			}
		}
	}

	if latitude, ok := extractFloat(raw, "latitude", "lat"); ok {
		l.Latitude = latitude
		l.HasLatitude = true
	} else if aux.Latitude != 0 || aux.Lat != 0 {
		if aux.Latitude != 0 {
			l.Latitude = aux.Latitude
		} else {
			l.Latitude = aux.Lat
		}
		l.HasLatitude = true
	}

	if longitude, ok := extractFloat(raw, "longitude", "lon"); ok {
		l.Longitude = longitude
		l.HasLongitude = true
	} else if aux.Longitude != 0 || aux.Lon != 0 {
		if aux.Longitude != 0 {
			l.Longitude = aux.Longitude
		} else {
			l.Longitude = aux.Lon
		}
		l.HasLongitude = true
	}

	if _, ok := raw["success"]; ok {
		success := aux.Success
		l.Success = &success
	}

	l.Status = aux.Status
	l.Message = aux.Message
	l.Error = aux.Error
	l.Reason = aux.Reason

	return nil
}

// GeoLookupConfig configures geolocation lookup behavior.
type GeoLookupConfig struct {
	ProviderURLTemplate string
	ProviderName        string
	Timeout             time.Duration
	CacheTTL            time.Duration
	Logger              *zap.Logger
	HTTPClient          *http.Client
}

type geoCacheEntry struct {
	location *LocationInfo
	at       time.Time
}

// GeoLookup resolves IP addresses to geolocation metadata.
type GeoLookup struct {
	providerURLTemplate string
	providerName        string
	cacheTTL            time.Duration
	client              *http.Client
	logger              *zap.Logger

	mu    sync.RWMutex
	cache map[string]geoCacheEntry
}

// NewGeoLookup creates a geolocation lookup with sensible defaults.
func NewGeoLookup(cfg GeoLookupConfig) *GeoLookup {
	providerURL := strings.TrimSpace(cfg.ProviderURLTemplate)
	if providerURL == "" {
		providerURL = strings.TrimSpace(os.Getenv(geoProviderURLEnv))
	}
	if providerURL == "" {
		providerURL = defaultGeoProviderURLTemplate
	}

	providerName := strings.TrimSpace(cfg.ProviderName)
	if providerName == "" {
		providerName = strings.TrimSpace(os.Getenv(geoProviderNameEnv))
	}
	if providerName == "" {
		providerName = defaultGeoProviderName
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultGeoTimeout
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultGeoCacheTTL
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	return &GeoLookup{
		providerURLTemplate: providerURL,
		providerName:        providerName,
		cacheTTL:            cacheTTL,
		client:              client,
		logger:              cfg.Logger,
		cache:               make(map[string]geoCacheEntry),
	}
}

// Lookup resolves geolocation for an IP address.
func (g *GeoLookup) Lookup(ipAddress string) *LocationInfo {
	if g == nil {
		return nil
	}
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return nil
	}

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		g.warn("invalid IP address format for geolocation", zap.String("ip", ipAddress))
		return nil
	}
	if !shouldLookupIP(ip) {
		g.debug("skipping geolocation for non-public ip", zap.String("ip", ipAddress))
		return nil
	}

	if cached, ok := g.getCached(ipAddress); ok {
		return cloneLocation(cached)
	}

	location := g.fetchFromProvider(ipAddress)
	g.setCached(ipAddress, location)
	return cloneLocation(location)
}

// LookupAsync resolves geolocation asynchronously and invokes callback.
func (g *GeoLookup) LookupAsync(ipAddress string, callback func(*LocationInfo)) {
	if callback == nil {
		g.error("lookup async called with nil callback", zap.String("ip", ipAddress))
		return
	}

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				g.error("panic recovered in async geolocation lookup", zap.Any("panic", recovered), zap.String("ip", ipAddress))
				callback(nil)
			}
		}()
		callback(g.Lookup(ipAddress))
	}()
}

// ClearCache clears cached geolocation responses.
func (g *GeoLookup) ClearCache() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cache = make(map[string]geoCacheEntry)
}

// CacheSize returns the number of cached IP entries.
func (g *GeoLookup) CacheSize() int {
	if g == nil {
		return 0
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.cache)
}

func (g *GeoLookup) getCached(ipAddress string) (*LocationInfo, bool) {
	g.mu.RLock()
	entry, ok := g.cache[ipAddress]
	g.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if time.Since(entry.at) > g.cacheTTL {
		g.mu.Lock()
		delete(g.cache, ipAddress)
		g.mu.Unlock()
		return nil, false
	}

	return entry.location, true
}

func (g *GeoLookup) setCached(ipAddress string, location *LocationInfo) {
	g.mu.Lock()
	g.cache[ipAddress] = geoCacheEntry{
		location: cloneLocation(location),
		at:       time.Now(),
	}
	g.mu.Unlock()
}

func (g *GeoLookup) fetchFromProvider(ipAddress string) *LocationInfo {
	url := fmt.Sprintf(g.providerURLTemplate, ipAddress)
	resp, err := g.client.Get(url)
	if err != nil {
		g.warn("failed to fetch geolocation", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		g.warn("geolocation provider returned non-200", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.Int("status", resp.StatusCode))
		return nil
	}

	var location LocationInfo
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		g.warn("failed to decode geolocation response", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.Error(err))
		return nil
	}

	if location.Success != nil && !*location.Success {
		g.debug("geolocation provider reported unsuccessful lookup", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.String("message", location.Message))
		return nil
	}
	if location.Error {
		g.debug("geolocation provider reported error", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.String("reason", location.Reason))
		return nil
	}
	if location.Status != "" && !strings.EqualFold(location.Status, "success") {
		g.debug("geolocation provider reported failure status", zap.String("provider", g.providerName), zap.String("ip", ipAddress), zap.String("status", location.Status), zap.String("message", location.Message))
		return nil
	}

	return &location
}

func shouldLookupIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsMulticast() &&
		!ip.IsUnspecified()
}

func cloneLocation(location *LocationInfo) *LocationInfo {
	if location == nil {
		return nil
	}
	copyLocation := *location
	if location.Success != nil {
		success := *location.Success
		copyLocation.Success = &success
	}
	return &copyLocation
}

func extractFloat(raw map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case json.Number:
			if parsed, err := v.Float64(); err == nil {
				return parsed, true
			}
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func (g *GeoLookup) debug(msg string, fields ...zap.Field) {
	if g != nil && g.logger != nil {
		g.logger.Debug(msg, fields...)
	}
}

func (g *GeoLookup) warn(msg string, fields ...zap.Field) {
	if g != nil && g.logger != nil {
		g.logger.Warn(msg, fields...)
	}
}

func (g *GeoLookup) error(msg string, fields ...zap.Field) {
	if g != nil && g.logger != nil {
		g.logger.Error(msg, fields...)
	}
}
