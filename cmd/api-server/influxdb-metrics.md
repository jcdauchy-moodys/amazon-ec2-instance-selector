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
memory_per_vcpu (float) - Memory in GiB per vCPU (ratio)
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
instance_storage_gb (int) - Total instance storage size in GB
instance_storage_disks (int) - Number of instance storage disk types
disk_0_size_gb (int) - Size of first disk type in GB (per disk)
disk_0_count (int) - Number of disks of first type
disk_0_type (string) - Type of first disk (hdd, ssd)
disk_1_size_gb (int) - Size of second disk type in GB (if present)
disk_1_count (int) - Number of disks of second type (if present)
disk_1_type (string) - Type of second disk (if present)
nvme_support (string) - NVMe support level (required, supported, unsupported)
nvme_instance_storage_gb (int) - NVMe instance storage in GB (only when nvme_support="required")
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

## Example Queries

### Find memory-optimized instances (high memory per vCPU)
```sql
SELECT instance_type, memory_per_vcpu, memory_gb, vcpus 
FROM ec2_instances 
WHERE region='eu-west-1' 
  AND memory_per_vcpu > 8.0 
  AND current_generation=true 
ORDER BY memory_per_vcpu DESC 
LIMIT 10
```

### Find compute-optimized instances (low memory per vCPU)
```sql
SELECT instance_type, memory_per_vcpu, memory_gb, vcpus 
FROM ec2_instances 
WHERE region='eu-west-1' 
  AND memory_per_vcpu < 2.0 
  AND current_generation=true 
ORDER BY memory_per_vcpu ASC 
LIMIT 10
```

### Find balanced instances (moderate memory per vCPU)
```sql
SELECT instance_type, memory_per_vcpu, memory_gb, vcpus 
FROM ec2_instances 
WHERE region='us-east-1' 
  AND memory_per_vcpu >= 3.0 
  AND memory_per_vcpu <= 5.0 
  AND current_generation=true 
ORDER BY vcpus ASC
```

### Average memory per vCPU by instance family
```sql
SELECT family, MEAN(memory_per_vcpu) as avg_memory_per_vcpu, COUNT(*) as instance_count
FROM ec2_instances 
WHERE region='us-east-1' 
  AND current_generation=true 
GROUP BY family 
ORDER BY avg_memory_per_vcpu DESC
```

### Cost efficiency: Find instances with best price per GiB of memory
```sql
SELECT instance_type, 
       ondemand_price_per_hour, 
       memory_gb, 
       memory_per_vcpu,
       (ondemand_price_per_hour / memory_gb) as price_per_gb_memory
FROM ec2_instances 
WHERE region='us-east-1' 
  AND ondemand_price_per_hour > 0 
  AND current_generation=true 
ORDER BY price_per_gb_memory ASC 
LIMIT 20
```

### Find instances with NVMe instance storage
```sql
SELECT instance_type, 
       nvme_instance_storage_gb,
       instance_storage_disks,
       disk_0_size_gb,
       disk_0_count,
       disk_0_type,
       nvme_support
FROM ec2_instances 
WHERE region='us-east-1' 
  AND nvme_support='required'
  AND current_generation=true
ORDER BY nvme_instance_storage_gb DESC
LIMIT 20
```

### Find instances with SSD storage (NVMe or not)
```sql
SELECT instance_type, 
       instance_storage_gb, 
       disk_0_size_gb,
       disk_0_count,
       disk_0_type,
       nvme_support
FROM ec2_instances 
WHERE region='us-east-1' 
  AND disk_0_type='ssd'
  AND current_generation=true
ORDER BY instance_storage_gb DESC
```

### Calculate total NVMe storage for multi-disk instances
```sql
SELECT instance_type,
       nvme_instance_storage_gb as total_nvme_storage,
       (disk_0_size_gb * disk_0_count) as disk_0_total,
       (disk_1_size_gb * disk_1_count) as disk_1_total,
       disk_0_type,
       nvme_support
FROM ec2_instances 
WHERE region='us-east-1' 
  AND instance_storage_disks > 1
  AND nvme_support='required'
ORDER BY total_nvme_storage DESC
LIMIT 20
```

### Find NVMe instances with storage greater than 1TB
```sql
SELECT instance_type, 
       nvme_instance_storage_gb,
       memory_gb,
       vcpus,
       ondemand_price_per_hour
FROM ec2_instances 
WHERE region='us-east-1' 
  AND nvme_instance_storage_gb > 1000
  AND current_generation=true
ORDER BY nvme_instance_storage_gb ASC
```

### Price per TB of NVMe storage analysis
```sql
SELECT instance_type,
       nvme_instance_storage_gb,
       ondemand_price_per_hour,
       (ondemand_price_per_hour / (nvme_instance_storage_gb / 1000.0)) as price_per_tb_nvme
FROM ec2_instances 
WHERE region='us-east-1' 
  AND nvme_instance_storage_gb > 0
  AND ondemand_price_per_hour > 0
  AND current_generation=true
ORDER BY price_per_tb_nvme ASC
LIMIT 20
```