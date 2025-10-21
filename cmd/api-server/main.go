// EC2 Instance Selector API Server
//
// This API server provides REST endpoints for filtering EC2 instance types.
//
// Environment Variables:
//
//	EC2_INSTANCE_SELECTOR_CACHE_TTL - Cache time-to-live for pricing data (default: 24h)
//	                                   Examples: "1h", "30m", "24h", "0" (disables cache)
//	EC2_INSTANCE_SELECTOR_CACHE_DIR - Directory for cache files (default: ~/.ec2-instance-selector/)
//	EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT - Skip pricing cache initialization on startup (default: false)
//	                                   Set to "true" for faster startup without pricing data
//	EC2_INSTANCE_SELECTOR_VERBOSE   - Enable verbose/debug logging (default: false)
//	                                   Set to "true" to see detailed AWS API calls and timing information
//	PORT                            - Server port (default: 8080)
//	AWS_REGION or AWS_DEFAULT_REGION - AWS region (required)
//	INFLUXDB_ENABLED                - Enable InfluxDB metrics collection (default: false)
//	INFLUXDB_URL                    - InfluxDB server URL (required if metrics enabled)
//	INFLUXDB_DATABASE               - InfluxDB database name (required if metrics enabled)
//	INFLUXDB_JWT                    - JWT token for InfluxDB authentication (optional)
//
// Example:
//
//	export EC2_INSTANCE_SELECTOR_CACHE_TTL=12h
//	export EC2_INSTANCE_SELECTOR_CACHE_DIR=/tmp/ec2-cache/
//	export PORT=3000
//	./api-server
//
// Example (fast startup without pricing):
//
//	export EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT=true
//	./api-server
//
// Example (with InfluxDB metrics):
//
//	export INFLUXDB_ENABLED=true
//	export INFLUXDB_URL=https://influxdb.example.com
//	export INFLUXDB_DATABASE=ec2metrics
//	export INFLUXDB_JWT=your-jwt-token-here
//	./api-server
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/bytequantity"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/instancetypes"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/metrics"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/selector"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mitchellh/go-homedir"
)

type APIServerConfig struct {
	CacheTTL             time.Duration
	CacheDir             string
	Port                 string
	SkipPricingCacheInit bool
	Verbose              bool
	InfluxDB             metrics.InfluxDBConfig
}

type APIServer struct {
	selector      *selector.Selector
	metricsClient *metrics.InfluxDBClient
}

type FilterRequest struct {
	VCPUs              *int32   `json:"vcpus,omitempty"`
	VCPUsMin           *int32   `json:"vcpus_min,omitempty"`
	VCPUsMax           *int32   `json:"vcpus_max,omitempty"`
	Memory             *string  `json:"memory,omitempty"`
	MemoryMin          *string  `json:"memory_min,omitempty"`
	MemoryMax          *string  `json:"memory_max,omitempty"`
	MemoryPerCpuMin    *float64 `json:"memory_per_cpu_min,omitempty"`
	MemoryPerCpuMax    *float64 `json:"memory_per_cpu_max,omitempty"`
	CPUArchitecture    *string  `json:"cpu_architecture,omitempty"`
	InstanceTypes      []string `json:"instance_types,omitempty"`
	AllowList          *string  `json:"allow_list,omitempty"`
	DenyList           *string  `json:"deny_list,omitempty"`
	CurrentGeneration  *bool    `json:"current_generation,omitempty"`
	BareMetal          *bool    `json:"bare_metal,omitempty"`
	Burstable          *bool    `json:"burstable,omitempty"`
	MaxResults         *int     `json:"max_results,omitempty"`
	Region             *string  `json:"region,omitempty"`
	AvailabilityZones  []string `json:"availability_zones,omitempty"`
	UsageClass         *string  `json:"usage_class,omitempty"`
	GPUs               *int32   `json:"gpus,omitempty"`
	GPUsMin            *int32   `json:"gpus_min,omitempty"`
	GPUsMax            *int32   `json:"gpus_max,omitempty"`
	NetworkPerformance *int     `json:"network_performance,omitempty"`
	FreeTier           *bool    `json:"free_tier,omitempty"`
	NVME               *bool    `json:"nvme,omitempty"`
}

type APIResponse struct {
	Success       bool                     `json:"success"`
	Message       string                   `json:"message,omitempty"`
	InstanceTypes []*instancetypes.Details `json:"instance_types,omitempty"`
	Count         int                      `json:"count"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func NewAPIServer(serverConfig APIServerConfig) (*APIServer, error) {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Validate that region is set
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region must be configured. Set AWS_REGION or AWS_DEFAULT_REGION environment variable, or configure it in ~/.aws/config")
	}

	log.Printf("Using AWS region: %s", cfg.Region)

	var expandedCacheDir string
	if serverConfig.CacheTTL > 0 {
		// Ensure cache directory exists only if caching is enabled
		expandedCacheDir, err = ensureCacheDir(serverConfig.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to setup cache directory: %w", err)
		}
	} else {
		log.Printf("Caching disabled (TTL = 0), pricing data will not be cached")
		expandedCacheDir = serverConfig.CacheDir
	}

	// Create selector instance with configurable pricing cache
	log.Printf("Initializing selector with cache TTL: %v, cache dir: %s", serverConfig.CacheTTL, expandedCacheDir)
	instanceSelector, err := selector.NewWithCache(ctx, cfg, serverConfig.CacheTTL, expandedCacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create selector: %w", err)
	}

	// Enable detailed logging for the selector if verbose mode is on
	if serverConfig.Verbose {
		log.Printf("Verbose logging enabled")
		instanceSelector.SetLogger(log.Default())
	} else {
		// Disable verbose selector logs in normal mode
		instanceSelector.SetLogger(log.New(io.Discard, "", 0))
	}

	// Initialize pricing caches
	if !serverConfig.SkipPricingCacheInit {
		if err := initializePricingCaches(ctx, instanceSelector, serverConfig.CacheTTL); err != nil {
			return nil, fmt.Errorf("failed to initialize pricing caches: %w", err)
		}
	} else {
		log.Printf("Skipping pricing cache initialization (SKIP_PRICING_CACHE_INIT=true)")
	}

	// Initialize InfluxDB client if enabled
	var metricsClient *metrics.InfluxDBClient
	if serverConfig.InfluxDB.Enabled {
		metricsClient, err = metrics.NewInfluxDBClient(serverConfig.InfluxDB)
		if err != nil {
			return nil, fmt.Errorf("failed to create InfluxDB client: %w", err)
		}
		log.Printf("InfluxDB metrics collection enabled")
	}

	return &APIServer{
		selector:      instanceSelector,
		metricsClient: metricsClient,
	}, nil
}

func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) filterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding JSON request: %v", err)
		s.sendError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Convert request to selector filters
	filters, err := s.requestToFilters(req)
	if err != nil {
		log.Printf("Error converting request to filters: %v", err)
		s.sendError(w, fmt.Sprintf("Invalid filter parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Execute the filter
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	instanceTypes, err := s.selector.FilterVerbose(ctx, filters)
	if err != nil {
		log.Printf("Filter execution failed: %v", err)
		s.sendError(w, fmt.Sprintf("Filter execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Filter executed in %v, found %d instances", time.Since(start), len(instanceTypes))

	// Record metrics if enabled
	if s.metricsClient != nil {
		region := "unknown"
		if filters.Region != nil {
			region = *filters.Region
		}
		if err := s.metricsClient.RecordInstanceTypes(instanceTypes, region); err != nil {
			log.Printf("Warning: failed to record metrics: %v", err)
		}
	}

	// Limit results if max_results is specified
	maxResults := 20 // default
	if req.MaxResults != nil {
		maxResults = *req.MaxResults
	}

	if len(instanceTypes) > maxResults {
		instanceTypes = instanceTypes[:maxResults]
	}

	// Convert to response format
	response := APIResponse{
		Success:       true,
		InstanceTypes: instanceTypes,
		Count:         len(instanceTypes),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters into FilterRequest
	req := s.parseQueryParams(r)

	// Convert request to selector filters
	filters, err := s.requestToFilters(req)
	if err != nil {
		log.Printf("Error converting request to filters: %v", err)
		s.sendError(w, fmt.Sprintf("Invalid filter parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Execute the filter
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	instanceTypes, err := s.selector.FilterVerbose(ctx, filters)
	if err != nil {
		log.Printf("Filter execution failed: %v", err)
		s.sendError(w, fmt.Sprintf("Filter execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Filter executed in %v, found %d instances", time.Since(start), len(instanceTypes))

	// Limit results if max_results is specified
	maxResults := 20 // default
	if req.MaxResults != nil {
		maxResults = *req.MaxResults
	}

	if len(instanceTypes) > maxResults {
		instanceTypes = instanceTypes[:maxResults]
	}

	// Convert to response format
	response := APIResponse{
		Success:       true,
		InstanceTypes: instanceTypes,
		Count:         len(instanceTypes),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) parseQueryParams(r *http.Request) FilterRequest {
	req := FilterRequest{}

	if vcpus := r.URL.Query().Get("vcpus"); vcpus != "" {
		if v, err := strconv.ParseInt(vcpus, 10, 32); err == nil {
			val := int32(v)
			req.VCPUs = &val
		}
	}

	if vcpusMin := r.URL.Query().Get("vcpus_min"); vcpusMin != "" {
		if v, err := strconv.ParseInt(vcpusMin, 10, 32); err == nil {
			val := int32(v)
			req.VCPUsMin = &val
		}
	}

	if vcpusMax := r.URL.Query().Get("vcpus_max"); vcpusMax != "" {
		if v, err := strconv.ParseInt(vcpusMax, 10, 32); err == nil {
			val := int32(v)
			req.VCPUsMax = &val
		}
	}

	if memory := r.URL.Query().Get("memory"); memory != "" {
		req.Memory = &memory
	}

	if memoryMin := r.URL.Query().Get("memory_min"); memoryMin != "" {
		req.MemoryMin = &memoryMin
	}

	if memoryMax := r.URL.Query().Get("memory_max"); memoryMax != "" {
		req.MemoryMax = &memoryMax
	}

	if memoryPerCpuMin := r.URL.Query().Get("memory_per_cpu_min"); memoryPerCpuMin != "" {
		if v, err := strconv.ParseFloat(memoryPerCpuMin, 64); err == nil {
			req.MemoryPerCpuMin = &v
		}
	}

	if memoryPerCpuMax := r.URL.Query().Get("memory_per_cpu_max"); memoryPerCpuMax != "" {
		if v, err := strconv.ParseFloat(memoryPerCpuMax, 64); err == nil {
			req.MemoryPerCpuMax = &v
		}
	}

	if arch := r.URL.Query().Get("cpu_architecture"); arch != "" {
		req.CPUArchitecture = &arch
	}

	if instanceTypes := r.URL.Query().Get("instance_types"); instanceTypes != "" {
		req.InstanceTypes = strings.Split(instanceTypes, ",")
	}

	if allowList := r.URL.Query().Get("allow_list"); allowList != "" {
		req.AllowList = &allowList
	}

	if denyList := r.URL.Query().Get("deny_list"); denyList != "" {
		req.DenyList = &denyList
	}

	if currentGen := r.URL.Query().Get("current_generation"); currentGen != "" {
		if v, err := strconv.ParseBool(currentGen); err == nil {
			req.CurrentGeneration = &v
		}
	}

	if maxResults := r.URL.Query().Get("max_results"); maxResults != "" {
		if v, err := strconv.Atoi(maxResults); err == nil {
			req.MaxResults = &v
		}
	}

	if zones := r.URL.Query().Get("availability_zones"); zones != "" {
		req.AvailabilityZones = strings.Split(zones, ",")
	}

	if gpus := r.URL.Query().Get("gpus"); gpus != "" {
		if v, err := strconv.ParseInt(gpus, 10, 32); err == nil {
			val := int32(v)
			req.GPUs = &val
		}
	}

	if gpusMin := r.URL.Query().Get("gpus_min"); gpusMin != "" {
		if v, err := strconv.ParseInt(gpusMin, 10, 32); err == nil {
			val := int32(v)
			req.GPUsMin = &val
		}
	}

	if gpusMax := r.URL.Query().Get("gpus_max"); gpusMax != "" {
		if v, err := strconv.ParseInt(gpusMax, 10, 32); err == nil {
			val := int32(v)
			req.GPUsMax = &val
		}
	}

	if networkPerformance := r.URL.Query().Get("network_performance"); networkPerformance != "" {
		if v, err := strconv.Atoi(networkPerformance); err == nil {
			req.NetworkPerformance = &v
		}
	}

	if freeTier := r.URL.Query().Get("free_tier"); freeTier != "" {
		if v, err := strconv.ParseBool(freeTier); err == nil {
			req.FreeTier = &v
		}
	}

	if bareMetal := r.URL.Query().Get("bare_metal"); bareMetal != "" {
		if v, err := strconv.ParseBool(bareMetal); err == nil {
			req.BareMetal = &v
		}
	}

	if burstable := r.URL.Query().Get("burstable"); burstable != "" {
		if v, err := strconv.ParseBool(burstable); err == nil {
			req.Burstable = &v
		}
	}

	if usageClass := r.URL.Query().Get("usage_class"); usageClass != "" {
		req.UsageClass = &usageClass
	}

	if region := r.URL.Query().Get("region"); region != "" {
		req.Region = &region
	}

	if nvme := r.URL.Query().Get("nvme"); nvme != "" {
		if v, err := strconv.ParseBool(nvme); err == nil {
			req.NVME = &v
		}
	}

	return req
}

func (s *APIServer) requestToFilters(req FilterRequest) (selector.Filters, error) {
	filters := selector.Filters{}

	// VCPUs range
	if req.VCPUs != nil {
		filters.VCpusRange = &selector.Int32RangeFilter{
			LowerBound: *req.VCPUs,
			UpperBound: *req.VCPUs,
		}
	} else if req.VCPUsMin != nil || req.VCPUsMax != nil {
		rangeFilter := &selector.Int32RangeFilter{}
		if req.VCPUsMin != nil {
			rangeFilter.LowerBound = *req.VCPUsMin
		}
		if req.VCPUsMax != nil {
			rangeFilter.UpperBound = *req.VCPUsMax
		}
		filters.VCpusRange = rangeFilter
	}

	// Memory range
	if req.Memory != nil {
		bq, err := bytequantity.ParseToByteQuantity(*req.Memory)
		if err != nil {
			return filters, fmt.Errorf("invalid memory format: %w", err)
		}
		filters.MemoryRange = &selector.ByteQuantityRangeFilter{
			LowerBound: bq,
			UpperBound: bq,
		}
	} else if req.MemoryMin != nil || req.MemoryMax != nil {
		rangeFilter := &selector.ByteQuantityRangeFilter{}
		if req.MemoryMin != nil {
			bq, err := bytequantity.ParseToByteQuantity(*req.MemoryMin)
			if err != nil {
				return filters, fmt.Errorf("invalid memory_min format: %w", err)
			}
			rangeFilter.LowerBound = bq
		}
		if req.MemoryMax != nil {
			bq, err := bytequantity.ParseToByteQuantity(*req.MemoryMax)
			if err != nil {
				return filters, fmt.Errorf("invalid memory_max format: %w", err)
			}
			rangeFilter.UpperBound = bq
		}
		filters.MemoryRange = rangeFilter
	}

	// Memory per CPU range
	if req.MemoryPerCpuMin != nil || req.MemoryPerCpuMax != nil {
		rangeFilter := &selector.Float64RangeFilter{}
		if req.MemoryPerCpuMin != nil {
			rangeFilter.LowerBound = *req.MemoryPerCpuMin
		}
		if req.MemoryPerCpuMax != nil {
			rangeFilter.UpperBound = *req.MemoryPerCpuMax
		}
		filters.MemoryPerCpuRange = rangeFilter
	}

	// CPU Architecture
	if req.CPUArchitecture != nil {
		arch := ec2types.ArchitectureType(*req.CPUArchitecture)
		filters.CPUArchitecture = &arch
	}

	// Instance types
	if len(req.InstanceTypes) > 0 {
		instanceTypeStrs := make([]string, len(req.InstanceTypes))
		copy(instanceTypeStrs, req.InstanceTypes)
		filters.InstanceTypes = &instanceTypeStrs
	}

	// Allow/Deny lists - compile regex patterns
	if req.AllowList != nil {
		regex, err := regexp.Compile(*req.AllowList)
		if err != nil {
			return filters, fmt.Errorf("invalid allow_list regex: %w", err)
		}
		filters.AllowList = regex
	}
	if req.DenyList != nil {
		regex, err := regexp.Compile(*req.DenyList)
		if err != nil {
			return filters, fmt.Errorf("invalid deny_list regex: %w", err)
		}
		filters.DenyList = regex
	}

	// Boolean filters
	filters.CurrentGeneration = req.CurrentGeneration
	filters.BareMetal = req.BareMetal
	filters.Burstable = req.Burstable
	filters.FreeTier = req.FreeTier
	filters.NVME = req.NVME

	// Availability zones
	if len(req.AvailabilityZones) > 0 {
		zones := make([]string, len(req.AvailabilityZones))
		copy(zones, req.AvailabilityZones)
		filters.AvailabilityZones = &zones
	}

	// Usage class
	if req.UsageClass != nil {
		usageClass := ec2types.UsageClassType(*req.UsageClass)
		filters.UsageClass = &usageClass
	}

	// GPUs range
	if req.GPUs != nil {
		filters.GpusRange = &selector.Int32RangeFilter{
			LowerBound: *req.GPUs,
			UpperBound: *req.GPUs,
		}
	} else if req.GPUsMin != nil || req.GPUsMax != nil {
		rangeFilter := &selector.Int32RangeFilter{}
		if req.GPUsMin != nil {
			rangeFilter.LowerBound = *req.GPUsMin
		}
		if req.GPUsMax != nil {
			rangeFilter.UpperBound = *req.GPUsMax
		}
		filters.GpusRange = rangeFilter
	}

	// Network performance
	if req.NetworkPerformance != nil {
		filters.NetworkPerformance = &selector.IntRangeFilter{
			LowerBound: *req.NetworkPerformance,
			UpperBound: *req.NetworkPerformance,
		}
	}

	// Set region if provided
	if req.Region != nil {
		filters.Region = req.Region
	}

	return filters, nil
}

func (s *APIServer) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Message: message,
		Count:   0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func ensureCacheDir(cacheDir string) (string, error) {
	// Expand home directory path if needed
	expandedPath, err := homedir.Expand(cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to expand cache directory path %s: %w", cacheDir, err)
	}

	log.Printf("Cache directory (expanded): %s", expandedPath)

	// Check if directory exists
	if stat, err := os.Stat(expandedPath); err == nil {
		if !stat.IsDir() {
			return "", fmt.Errorf("cache path %s exists but is not a directory", expandedPath)
		}
		log.Printf("Cache directory already exists: %s", expandedPath)
		return expandedPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check cache directory %s: %w", expandedPath, err)
	}

	// Directory doesn't exist, create it
	log.Printf("Creating cache directory: %s", expandedPath)
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory %s: %w", expandedPath, err)
	}

	log.Printf("Successfully created cache directory: %s", expandedPath)
	return expandedPath, nil
}

func initializePricingCaches(ctx context.Context, instanceSelector *selector.Selector, ttl time.Duration) error {
	if ttl <= 0 {
		log.Printf("Pricing cache disabled (TTL = 0), skipping cache initialization")
		return nil
	}

	log.Printf("Initializing pricing caches...")

	// Check and refresh on-demand pricing cache
	onDemandCount := instanceSelector.EC2Pricing.OnDemandCacheCount()
	spotCount := instanceSelector.EC2Pricing.SpotCacheCount()

	log.Printf("Current cache counts - OnDemand: %d, Spot: %d", onDemandCount, spotCount)

	if onDemandCount == 0 {
		log.Printf("On-demand pricing cache is empty, refreshing...")
		if err := instanceSelector.EC2Pricing.RefreshOnDemandCache(ctx); err != nil {
			return fmt.Errorf("failed to refresh on-demand pricing cache: %w", err)
		}
		log.Printf("On-demand pricing cache refreshed successfully")
	} else {
		log.Printf("On-demand pricing cache already populated (%d entries)", onDemandCount)
	}

	if spotCount == 0 {
		log.Printf("Spot pricing cache is empty, refreshing...")
		if err := instanceSelector.EC2Pricing.RefreshSpotCache(ctx, 30); err != nil {
			log.Printf("Warning: failed to refresh spot pricing cache: %v", err)
			// Don't return error for spot pricing failures as it's less critical
		} else {
			log.Printf("Spot pricing cache refreshed successfully")
		}
	} else {
		log.Printf("Spot pricing cache already populated (%d entries)", spotCount)
	}

	// Log final cache counts
	finalOnDemandCount := instanceSelector.EC2Pricing.OnDemandCacheCount()
	finalSpotCount := instanceSelector.EC2Pricing.SpotCacheCount()
	log.Printf("Final cache counts - OnDemand: %d, Spot: %d", finalOnDemandCount, finalSpotCount)

	return nil
}

func parseConfig() APIServerConfig {
	return APIServerConfig{
		CacheTTL:             getEnvDuration("EC2_INSTANCE_SELECTOR_CACHE_TTL", 24*time.Hour),
		CacheDir:             getEnvString("EC2_INSTANCE_SELECTOR_CACHE_DIR", "~/.ec2-instance-selector/"),
		Port:                 getEnvString("PORT", "8080"),
		SkipPricingCacheInit: getEnvBool("EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT", false),
		Verbose:              getEnvBool("EC2_INSTANCE_SELECTOR_VERBOSE", false),
		InfluxDB: metrics.InfluxDBConfig{
			Enabled:  getEnvBool("INFLUXDB_ENABLED", false),
			URL:      getEnvString("INFLUXDB_URL", ""),
			Database: getEnvString("INFLUXDB_DATABASE", ""),
			JWT:      getEnvString("INFLUXDB_JWT", ""),
		},
	}
}

func main() {
	config := parseConfig()

	log.Printf("Starting EC2 Instance Selector API Server...")
	log.Printf("Configuration:")
	log.Printf("  Cache TTL: %v", config.CacheTTL)
	log.Printf("  Cache Directory (raw): %s", config.CacheDir)
	log.Printf("  Port: %s", config.Port)
	log.Printf("  Skip Pricing Cache Init: %t", config.SkipPricingCacheInit)
	log.Printf("  Verbose Logging: %t", config.Verbose)

	server, err := NewAPIServer(config)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Setup routes
	http.HandleFunc("/health", server.healthHandler)
	http.HandleFunc("/api/v1/instances/filter", server.filterHandler)
	http.HandleFunc("/api/v1/instances", server.getHandler)

	// Add CORS headers middleware
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	})

	port := ":" + config.Port
	log.Printf("Server successfully initialized!")
	log.Printf("Listening on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  GET  /health                     - Health check")
	log.Printf("  GET  /api/v1/instances           - Filter instances via query params")
	log.Printf("  POST /api/v1/instances/filter    - Filter instances via JSON body")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
