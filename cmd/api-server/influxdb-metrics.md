# InfluxDB Metrics Documentation

## Configuration

The API server can send EC2 instance metrics to InfluxDB v1.x using the following environment variables:

- `INFLUXDB_ENABLED` - Enable InfluxDB metrics collection (default: false)
- `INFLUXDB_URL` - InfluxDB server URL (required if enabled)
- `INFLUXDB_DATABASE` - InfluxDB database name (required if enabled)
- `INFLUXDB_JWT` - JWT token for InfluxDB authentication (optional)

When `INFLUXDB_JWT` is set, the server will include an `Authorization: Bearer <token>` header in all HTTP requests to InfluxDB.

## Health Check

On startup, the API server will test connectivity to InfluxDB by calling the `/health` endpoint. If the health check fails, a warning is logged but the server continues to run. This allows the application to start even if InfluxDB is temporarily unavailable.

## Logging

The following information is logged when InfluxDB metrics are enabled:

- InfluxDB URL and database name on startup
- JWT authentication status (enabled/disabled)
- Health check results (OK or failure)
- Number of metrics successfully written for each batch (e.g., "Successfully wrote 25 ec2_instances metrics to InfluxDB")

## Measurement Schema

Measurement
ec2_instances - Single measurement for all EC2 instance data

*** Tags (unchanged - for querying/filtering)
region - AWS region
instance_type - Full instance type (e.g., "m5.large")
family - Instance family (e.g., "m5")
size - Instance size (e.g., "large")
architecture - CPU architecture (e.g., "x86_64", "arm64")
Fields (greatly expanded - actual metrics)
** VCpu Information:
vcpus (int) - Number of vCPUs
cores (int) - Number of cores
threads_per_core (int) - Threads per core
** Memory Information:
memory_mib (int) - Memory in MiB
memory_gb (float) - Memory in GB
*** Processor Information:
cpu_manufacturer (string) - CPU manufacturer (e.g., "Intel", "AMD", "AWS")
cpu_clock_speed_ghz (float) - Sustained clock speed in GHz
*** Network Information:
max_network_interfaces (int) - Maximum network interfaces
network_performance (string) - Network performance description
ipv4_per_interface (int) - IPv4 addresses per interface
ipv6_per_interface (int) - IPv6 addresses per interface
ipv6_supported (bool) - IPv6 support
ena_support (string) - ENA support level
efa_supported (bool) - EFA support
network_encryption_supported (bool) - Network encryption support
network_cards (int) - Number of network cards
** GPU Information:
gpu_memory_mib (int) - Total GPU memory in MiB
gpu_memory_gb (float) - Total GPU memory in GB
gpu_count (int) - Total number of GPUs
** Instance Storage Information:
instance_storage_gb (int) - Instance storage size in GB
instance_storage_disks (int) - Number of instance storage disks
nvme_support (string) - NVME support level
storage_encryption_support (string) - Storage encryption support
** EBS Information:
ebs_optimized_support (string) - EBS optimized support level
ebs_encryption_support (string) - EBS encryption support
ebs_nvme_support (string) - EBS NVME support
ebs_baseline_bandwidth_mbps (int) - EBS baseline bandwidth
ebs_baseline_throughput_mbps (float) - EBS baseline throughput
ebs_baseline_iops (int) - EBS baseline IOPS
ebs_max_bandwidth_mbps (int) - EBS maximum bandwidth
ebs_max_throughput_mbps (float) - EBS maximum throughput
ebs_max_iops (int) - EBS maximum IOPS
** Instance Properties:
current_generation (bool) - Is current generation
free_tier_eligible (bool) - Free tier eligibility
hibernation_supported (bool) - Hibernation support
dedicated_hosts_supported (bool) - Dedicated hosts support
auto_recovery_supported (bool) - Auto recovery support
bare_metal (bool) - Is bare metal
burstable_performance (bool) - Burstable performance support
hypervisor (string) - Hypervisor type
** Pricing Information:
ondemand_price_per_hour (float) - On-demand price per hour
spot_price (float) - Spot price
** Other:
inference_accelerator_count (int) - Number of inference accelerators
placement_group_strategies (int) - Number of supported placement group strategies

All fields are optional and only written if the data is available, ensuring efficient storage in InfluxDB. The code properly handles nil checks and only includes fields that have actual values.