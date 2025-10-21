package metrics

import (
	"context"
	"fmt"
	"io"
	"log"
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
	JWT        string
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

		// Build tags (dimensions for querying/filtering)
		tags := fmt.Sprintf("region=%s,instance_type=%s,family=%s,size=%s,architecture=%s",
			region,
			instance.InstanceType,
			family,
			size,
			instance.ProcessorInfo.SupportedArchitectures[0])

		// Build fields (actual metric values)
		var fields []string

		// VCpu Information
		if instance.VCpuInfo.DefaultVCpus != nil {
			fields = append(fields, fmt.Sprintf("vcpus=%di", *instance.VCpuInfo.DefaultVCpus))
		}
		if instance.VCpuInfo.DefaultCores != nil {
			fields = append(fields, fmt.Sprintf("cores=%di", *instance.VCpuInfo.DefaultCores))
		}
		if instance.VCpuInfo.DefaultThreadsPerCore != nil {
			fields = append(fields, fmt.Sprintf("threads_per_core=%di", *instance.VCpuInfo.DefaultThreadsPerCore))
		}

		// Memory Information
		if instance.MemoryInfo.SizeInMiB != nil {
			fields = append(fields, fmt.Sprintf("memory_mib=%di", *instance.MemoryInfo.SizeInMiB))
			fields = append(fields, fmt.Sprintf("memory_gb=%f", float64(*instance.MemoryInfo.SizeInMiB)/1024.0))
		}

		// Processor Information
		if instance.ProcessorInfo.Manufacturer != nil {
			fields = append(fields, fmt.Sprintf("cpu_manufacturer=\"%s\"", *instance.ProcessorInfo.Manufacturer))
		}
		if instance.ProcessorInfo.SustainedClockSpeedInGhz != nil {
			fields = append(fields, fmt.Sprintf("cpu_clock_speed_ghz=%f", *instance.ProcessorInfo.SustainedClockSpeedInGhz))
		}

		// Network Information
		if instance.NetworkInfo != nil {
			if instance.NetworkInfo.MaximumNetworkInterfaces != nil {
				fields = append(fields, fmt.Sprintf("max_network_interfaces=%di", *instance.NetworkInfo.MaximumNetworkInterfaces))
			}
			if instance.NetworkInfo.NetworkPerformance != nil {
				fields = append(fields, fmt.Sprintf("network_performance=\"%s\"", *instance.NetworkInfo.NetworkPerformance))
			}
			if instance.NetworkInfo.Ipv4AddressesPerInterface != nil {
				fields = append(fields, fmt.Sprintf("ipv4_per_interface=%di", *instance.NetworkInfo.Ipv4AddressesPerInterface))
			}
			if instance.NetworkInfo.Ipv6AddressesPerInterface != nil {
				fields = append(fields, fmt.Sprintf("ipv6_per_interface=%di", *instance.NetworkInfo.Ipv6AddressesPerInterface))
			}
			if instance.NetworkInfo.Ipv6Supported != nil {
				fields = append(fields, fmt.Sprintf("ipv6_supported=%t", *instance.NetworkInfo.Ipv6Supported))
			}
			if instance.NetworkInfo.EnaSupport != "" {
				fields = append(fields, fmt.Sprintf("ena_support=\"%s\"", instance.NetworkInfo.EnaSupport))
			}
			if instance.NetworkInfo.EfaSupported != nil {
				fields = append(fields, fmt.Sprintf("efa_supported=%t", *instance.NetworkInfo.EfaSupported))
			}
			if instance.NetworkInfo.EncryptionInTransitSupported != nil {
				fields = append(fields, fmt.Sprintf("network_encryption_supported=%t", *instance.NetworkInfo.EncryptionInTransitSupported))
			}
			if len(instance.NetworkInfo.NetworkCards) > 0 {
				fields = append(fields, fmt.Sprintf("network_cards=%di", len(instance.NetworkInfo.NetworkCards)))
			}
		}

		// GPU Information
		if instance.GpuInfo != nil {
			if instance.GpuInfo.TotalGpuMemoryInMiB != nil {
				fields = append(fields, fmt.Sprintf("gpu_memory_mib=%di", *instance.GpuInfo.TotalGpuMemoryInMiB))
				fields = append(fields, fmt.Sprintf("gpu_memory_gb=%f", float64(*instance.GpuInfo.TotalGpuMemoryInMiB)/1024.0))
			}
			if len(instance.GpuInfo.Gpus) > 0 {
				totalGpuCount := int32(0)
				for _, gpu := range instance.GpuInfo.Gpus {
					if gpu.Count != nil {
						totalGpuCount += *gpu.Count
					}
				}
				fields = append(fields, fmt.Sprintf("gpu_count=%di", totalGpuCount))
			}
		}

		// Instance Storage Information
		if instance.InstanceStorageInfo != nil {
			if instance.InstanceStorageInfo.TotalSizeInGB != nil {
				fields = append(fields, fmt.Sprintf("instance_storage_gb=%di", *instance.InstanceStorageInfo.TotalSizeInGB))
			}
			if len(instance.InstanceStorageInfo.Disks) > 0 {
				fields = append(fields, fmt.Sprintf("instance_storage_disks=%di", len(instance.InstanceStorageInfo.Disks)))
			}
			if instance.InstanceStorageInfo.NvmeSupport != "" {
				fields = append(fields, fmt.Sprintf("nvme_support=\"%s\"", instance.InstanceStorageInfo.NvmeSupport))
			}
			if instance.InstanceStorageInfo.EncryptionSupport != "" {
				fields = append(fields, fmt.Sprintf("storage_encryption_support=\"%s\"", instance.InstanceStorageInfo.EncryptionSupport))
			}
		}

		// EBS Information
		if instance.EbsInfo != nil {
			if instance.EbsInfo.EbsOptimizedSupport != "" {
				fields = append(fields, fmt.Sprintf("ebs_optimized_support=\"%s\"", instance.EbsInfo.EbsOptimizedSupport))
			}
			if instance.EbsInfo.EncryptionSupport != "" {
				fields = append(fields, fmt.Sprintf("ebs_encryption_support=\"%s\"", instance.EbsInfo.EncryptionSupport))
			}
			if instance.EbsInfo.NvmeSupport != "" {
				fields = append(fields, fmt.Sprintf("ebs_nvme_support=\"%s\"", instance.EbsInfo.NvmeSupport))
			}
			if instance.EbsInfo.EbsOptimizedInfo != nil {
				if instance.EbsInfo.EbsOptimizedInfo.BaselineBandwidthInMbps != nil {
					fields = append(fields, fmt.Sprintf("ebs_baseline_bandwidth_mbps=%di", *instance.EbsInfo.EbsOptimizedInfo.BaselineBandwidthInMbps))
				}
				if instance.EbsInfo.EbsOptimizedInfo.BaselineThroughputInMBps != nil {
					fields = append(fields, fmt.Sprintf("ebs_baseline_throughput_mbps=%f", *instance.EbsInfo.EbsOptimizedInfo.BaselineThroughputInMBps))
				}
				if instance.EbsInfo.EbsOptimizedInfo.BaselineIops != nil {
					fields = append(fields, fmt.Sprintf("ebs_baseline_iops=%di", *instance.EbsInfo.EbsOptimizedInfo.BaselineIops))
				}
				if instance.EbsInfo.EbsOptimizedInfo.MaximumBandwidthInMbps != nil {
					fields = append(fields, fmt.Sprintf("ebs_max_bandwidth_mbps=%di", *instance.EbsInfo.EbsOptimizedInfo.MaximumBandwidthInMbps))
				}
				if instance.EbsInfo.EbsOptimizedInfo.MaximumThroughputInMBps != nil {
					fields = append(fields, fmt.Sprintf("ebs_max_throughput_mbps=%f", *instance.EbsInfo.EbsOptimizedInfo.MaximumThroughputInMBps))
				}
				if instance.EbsInfo.EbsOptimizedInfo.MaximumIops != nil {
					fields = append(fields, fmt.Sprintf("ebs_max_iops=%di", *instance.EbsInfo.EbsOptimizedInfo.MaximumIops))
				}
			}
		}

		// Instance Properties
		if instance.CurrentGeneration != nil {
			fields = append(fields, fmt.Sprintf("current_generation=%t", *instance.CurrentGeneration))
		}
		if instance.FreeTierEligible != nil {
			fields = append(fields, fmt.Sprintf("free_tier_eligible=%t", *instance.FreeTierEligible))
		}
		if instance.HibernationSupported != nil {
			fields = append(fields, fmt.Sprintf("hibernation_supported=%t", *instance.HibernationSupported))
		}
		if instance.DedicatedHostsSupported != nil {
			fields = append(fields, fmt.Sprintf("dedicated_hosts_supported=%t", *instance.DedicatedHostsSupported))
		}
		if instance.AutoRecoverySupported != nil {
			fields = append(fields, fmt.Sprintf("auto_recovery_supported=%t", *instance.AutoRecoverySupported))
		}
		if instance.BareMetal != nil {
			fields = append(fields, fmt.Sprintf("bare_metal=%t", *instance.BareMetal))
		}
		if instance.BurstablePerformanceSupported != nil {
			fields = append(fields, fmt.Sprintf("burstable_performance=%t", *instance.BurstablePerformanceSupported))
		}
		if instance.Hypervisor != "" {
			fields = append(fields, fmt.Sprintf("hypervisor=\"%s\"", instance.Hypervisor))
		}

		// Pricing Information
		if instance.OndemandPricePerHour != nil {
			fields = append(fields, fmt.Sprintf("ondemand_price_per_hour=%f", *instance.OndemandPricePerHour))
		}
		if instance.SpotPrice != nil {
			fields = append(fields, fmt.Sprintf("spot_price=%f", *instance.SpotPrice))
		}

		// Inference Accelerator Information
		if instance.InferenceAcceleratorInfo != nil && instance.InferenceAcceleratorInfo.Accelerators != nil {
			totalAcceleratorCount := int32(0)
			for _, acc := range instance.InferenceAcceleratorInfo.Accelerators {
				if acc.Count != nil {
					totalAcceleratorCount += *acc.Count
				}
			}
			if totalAcceleratorCount > 0 {
				fields = append(fields, fmt.Sprintf("inference_accelerator_count=%di", totalAcceleratorCount))
			}
		}

		// Placement Group Support
		if instance.PlacementGroupInfo != nil && instance.PlacementGroupInfo.SupportedStrategies != nil {
			fields = append(fields, fmt.Sprintf("placement_group_strategies=%di", len(instance.PlacementGroupInfo.SupportedStrategies)))
		}

		// Create the line protocol entry
		if len(fields) > 0 {
			line := fmt.Sprintf("ec2_instances,%s %s %d",
				tags,
				strings.Join(fields, ","),
				timestamp)
			lines = append(lines, line)
		}
	}

	// Send data to InfluxDB
	if len(lines) > 0 {
		if err := c.sendData(strings.Join(lines, "\n")); err != nil {
			return err
		}
		log.Printf("Successfully wrote %d ec2_instances metrics to InfluxDB", len(lines))
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

	// Add Authorization Bearer header if JWT is configured
	if c.config.JWT != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.JWT))
	}

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

// TestConnection tests the connection to InfluxDB by checking the /health endpoint
func (c *InfluxDBClient) TestConnection(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	healthURL := fmt.Sprintf("%s/health", strings.TrimRight(c.config.URL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add Authorization Bearer header if JWT is configured
	if c.config.JWT != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.JWT))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach InfluxDB health endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for better error messages
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("InfluxDB health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
