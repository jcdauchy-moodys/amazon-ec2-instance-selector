package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/instancetypes"
)

// InfluxDBConfig holds the configuration for InfluxDB connection
type InfluxDBConfig struct {
	Enabled    bool
	URL        string
	Database   string
	BatchSize  int
	FlushTimer time.Duration
}

// InfluxDBClient handles metrics collection and sending to InfluxDB
type InfluxDBClient struct {
	config InfluxDBConfig
	client *http.Client
}

// NewInfluxDBClient creates a new InfluxDB client
func NewInfluxDBClient(config InfluxDBConfig) (*InfluxDBClient, error) {
	if config.Enabled {
		if config.URL == "" {
			return nil, fmt.Errorf("InfluxDB URL is required when metrics are enabled")
		}
		if config.Database == "" {
			return nil, fmt.Errorf("InfluxDB database is required when metrics are enabled")
		}
	}

	// Set defaults if not provided
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.FlushTimer == 0 {
		config.FlushTimer = 10 * time.Second
	}

	return &InfluxDBClient{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// RecordInstanceTypes records metrics for filtered instance types
func (c *InfluxDBClient) RecordInstanceTypes(instances []*instancetypes.Details, region string) error {
	if !c.config.Enabled {
		return nil
	}

	// Build line protocol data
	var lines []string
	timestamp := time.Now().UnixNano()

	for _, instance := range instances {
		// Extract instance family/size
		parts := strings.Split(string(instance.InstanceType), ".")
		if len(parts) != 2 {
			continue
		}
		family := parts[0]
		size := parts[1]

		// Create a single measurement per instance
		line := fmt.Sprintf("ec2_instances,region=%s,instance_type=%s,family=%s,size=%s,architecture=%s vcpus=%di,memory_gb=%f %d",
			region,
			instance.InstanceType,
			family,
			size,
			instance.ProcessorInfo.SupportedArchitectures[0],
			instance.VCpuInfo.DefaultVCpus,
			float64(*instance.MemoryInfo.SizeInMiB)/1024.0,
			timestamp)

		lines = append(lines, line)
	}

	// Send data to InfluxDB
	if len(lines) > 0 {
		return c.sendData(strings.Join(lines, "\n"))
	}

	return nil
}

// sendData sends metrics data to InfluxDB using line protocol
func (c *InfluxDBClient) sendData(data string) error {
	url := fmt.Sprintf("%s/write?db=%s&precision=ns",
		strings.TrimRight(c.config.URL, "/"),
		c.config.Database)

	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to write metrics, status code: %d", resp.StatusCode)
	}

	return nil
}
