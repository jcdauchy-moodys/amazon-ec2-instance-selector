// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package selector

import (
	"encoding/json"
	"log"
	"regexp"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/awsapi"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/bytequantity"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/ec2pricing"
	"github.com/aws/amazon-ec2-instance-selector/v3/pkg/instancetypes"
)

// InstanceTypesOutput can be implemented to provide custom output to instance type results.
type InstanceTypesOutput interface {
	Output([]*instancetypes.Details) []string
}

// InstanceTypesOutputFn is the func type definition for InstanceTypesOuput.
type InstanceTypesOutputFn func([]*instancetypes.Details) []string

// Output implements InstanceTypesOutput interface on InstanceTypesOutputFn
// This allows any InstanceTypesOutputFn to be passed into funcs accepting InstanceTypesOutput interface.
func (fn InstanceTypesOutputFn) Output(instanceTypes []*instancetypes.Details) []string {
	return fn(instanceTypes)
}

// Selector is used to filter instance type resource specs.
type Selector struct {
	EC2                   awsapi.SelectorInterface
	EC2Pricing            ec2pricing.EC2PricingIface
	InstanceTypesProvider *instancetypes.Provider
	ServiceRegistry       ServiceRegistry
	Logger                *log.Logger
}

// IntRangeFilter holds an upper and lower bound int
// The lower and upper bound are used to range filter resource specs.
type IntRangeFilter struct {
	UpperBound int
	LowerBound int
}

// Int32RangeFilter holds an upper and lower bound int
// The lower and upper bound are used to range filter resource specs.
type Int32RangeFilter struct {
	UpperBound int32
	LowerBound int32
}

// Uint64RangeFilter holds an upper and lower bound uint64
// The lower and upper bound are used to range filter resource specs.
type Uint64RangeFilter struct {
	UpperBound uint64
	LowerBound uint64
}

// ByteQuantityRangeFilter holds an upper and lower bound byte quantity
// The lower and upper bound are used to range filter resource specs.
type ByteQuantityRangeFilter struct {
	UpperBound bytequantity.ByteQuantity
	LowerBound bytequantity.ByteQuantity
}

// Float64RangeFilter holds an upper and lower bound float64
// The lower and upper bound are used to range filter resource specs.
type Float64RangeFilter struct {
	UpperBound float64
	LowerBound float64
}

// filterPair holds a tuple of the passed in filter value and the instance resource spec value.
type filterPair struct {
	filterValue  interface{}
	instanceSpec interface{}
}

func getRegexpString(r *regexp.Regexp) *string {
	if r == nil {
		return nil
	}
	rStr := r.String()
	return &rStr
}

// MarshalIndent is used to return a pretty-print json representation of a Filters struct.
func (f *Filters) MarshalIndent(prefix, indent string) ([]byte, error) {
	type Alias Filters
	return json.MarshalIndent(&struct {
		AllowList *string
		DenyList  *string
		*Alias
	}{
		AllowList: getRegexpString(f.AllowList),
		DenyList:  getRegexpString(f.DenyList),
		Alias:     (*Alias)(f),
	}, prefix, indent)
}

// MarshalIndentOnlySetFilters returns a JSON representation with only non-nil/non-zero filters.
func (f *Filters) MarshalIndentOnlySetFilters(prefix, indent string) ([]byte, error) {
	result := make(map[string]interface{})

	if f.AvailabilityZones != nil && len(*f.AvailabilityZones) > 0 {
		result["AvailabilityZones"] = *f.AvailabilityZones
	}
	if f.BareMetal != nil {
		result["BareMetal"] = *f.BareMetal
	}
	if f.Burstable != nil {
		result["Burstable"] = *f.Burstable
	}
	if f.AutoRecovery != nil {
		result["AutoRecovery"] = *f.AutoRecovery
	}
	if f.FreeTier != nil {
		result["FreeTier"] = *f.FreeTier
	}
	if f.CPUArchitecture != nil {
		result["CPUArchitecture"] = string(*f.CPUArchitecture)
	}
	if f.CPUManufacturer != nil {
		result["CPUManufacturer"] = string(*f.CPUManufacturer)
	}
	if f.CurrentGeneration != nil {
		result["CurrentGeneration"] = *f.CurrentGeneration
	}
	if f.EnaSupport != nil {
		result["EnaSupport"] = *f.EnaSupport
	}
	if f.EfaSupport != nil {
		result["EfaSupport"] = *f.EfaSupport
	}
	if f.Fpga != nil {
		result["Fpga"] = *f.Fpga
	}
	if f.GpusRange != nil {
		result["GpusRange"] = map[string]interface{}{
			"LowerBound": f.GpusRange.LowerBound,
			"UpperBound": f.GpusRange.UpperBound,
		}
	}
	if f.GpuMemoryRange != nil {
		result["GpuMemoryRange"] = map[string]interface{}{
			"LowerBound": f.GpuMemoryRange.LowerBound.Quantity,
			"UpperBound": f.GpuMemoryRange.UpperBound.Quantity,
		}
	}
	if f.GPUManufacturer != nil {
		result["GPUManufacturer"] = *f.GPUManufacturer
	}
	if f.GPUModel != nil {
		result["GPUModel"] = *f.GPUModel
	}
	if f.InferenceAcceleratorsRange != nil {
		result["InferenceAcceleratorsRange"] = map[string]interface{}{
			"LowerBound": f.InferenceAcceleratorsRange.LowerBound,
			"UpperBound": f.InferenceAcceleratorsRange.UpperBound,
		}
	}
	if f.InferenceAcceleratorManufacturer != nil {
		result["InferenceAcceleratorManufacturer"] = *f.InferenceAcceleratorManufacturer
	}
	if f.InferenceAcceleratorModel != nil {
		result["InferenceAcceleratorModel"] = *f.InferenceAcceleratorModel
	}
	if f.HibernationSupported != nil {
		result["HibernationSupported"] = *f.HibernationSupported
	}
	if f.Hypervisor != nil {
		result["Hypervisor"] = string(*f.Hypervisor)
	}
	if f.MaxResults != nil {
		result["MaxResults"] = *f.MaxResults
	}
	if f.MemoryRange != nil {
		result["MemoryRange"] = map[string]interface{}{
			"LowerBound": f.MemoryRange.LowerBound.Quantity,
			"UpperBound": f.MemoryRange.UpperBound.Quantity,
		}
	}
	if f.NetworkInterfaces != nil {
		result["NetworkInterfaces"] = map[string]interface{}{
			"LowerBound": f.NetworkInterfaces.LowerBound,
			"UpperBound": f.NetworkInterfaces.UpperBound,
		}
	}
	if f.NetworkPerformance != nil {
		result["NetworkPerformance"] = map[string]interface{}{
			"LowerBound": f.NetworkPerformance.LowerBound,
			"UpperBound": f.NetworkPerformance.UpperBound,
		}
	}
	if f.NetworkEncryption != nil {
		result["NetworkEncryption"] = *f.NetworkEncryption
	}
	if f.IPv6 != nil {
		result["IPv6"] = *f.IPv6
	}
	if f.PlacementGroupStrategy != nil {
		result["PlacementGroupStrategy"] = *f.PlacementGroupStrategy
	}
	if f.Region != nil {
		result["Region"] = *f.Region
	}
	if f.RootDeviceType != nil {
		result["RootDeviceType"] = string(*f.RootDeviceType)
	}
	if f.UsageClass != nil {
		result["UsageClass"] = string(*f.UsageClass)
	}
	if f.VCpusRange != nil {
		result["VCpusRange"] = map[string]interface{}{
			"LowerBound": f.VCpusRange.LowerBound,
			"UpperBound": f.VCpusRange.UpperBound,
		}
	}
	if f.VCpusToMemoryRatio != nil {
		result["VCpusToMemoryRatio"] = *f.VCpusToMemoryRatio
	}
	if f.MemoryPerCpuRange != nil {
		result["MemoryPerCpuRange"] = map[string]interface{}{
			"LowerBound": f.MemoryPerCpuRange.LowerBound,
			"UpperBound": f.MemoryPerCpuRange.UpperBound,
		}
	}
	if f.AllowList != nil {
		result["AllowList"] = f.AllowList.String()
	}
	if f.DenyList != nil {
		result["DenyList"] = f.DenyList.String()
	}
	if f.InstanceTypeBase != nil {
		result["InstanceTypeBase"] = *f.InstanceTypeBase
	}
	if f.Flexible != nil {
		result["Flexible"] = *f.Flexible
	}
	if f.Service != nil {
		result["Service"] = *f.Service
	}
	if f.InstanceTypes != nil && len(*f.InstanceTypes) > 0 {
		result["InstanceTypes"] = *f.InstanceTypes
	}
	if f.VirtualizationType != nil {
		result["VirtualizationType"] = string(*f.VirtualizationType)
	}
	if f.PricePerHour != nil {
		result["PricePerHour"] = map[string]interface{}{
			"LowerBound": f.PricePerHour.LowerBound,
			"UpperBound": f.PricePerHour.UpperBound,
		}
	}
	if f.InstanceStorageRange != nil {
		result["InstanceStorageRange"] = map[string]interface{}{
			"LowerBound": f.InstanceStorageRange.LowerBound.Quantity,
			"UpperBound": f.InstanceStorageRange.UpperBound.Quantity,
		}
	}
	if f.NVMEInstanceStorageRange != nil {
		result["NVMEInstanceStorageRange"] = map[string]interface{}{
			"LowerBound": f.NVMEInstanceStorageRange.LowerBound.Quantity,
			"UpperBound": f.NVMEInstanceStorageRange.UpperBound.Quantity,
		}
	}
	if f.DiskType != nil {
		result["DiskType"] = *f.DiskType
	}
	if f.NVME != nil {
		result["NVME"] = *f.NVME
	}
	if f.EBSOptimized != nil {
		result["EBSOptimized"] = *f.EBSOptimized
	}
	if f.DiskEncryption != nil {
		result["DiskEncryption"] = *f.DiskEncryption
	}
	if f.EBSOptimizedBaselineBandwidth != nil {
		result["EBSOptimizedBaselineBandwidth"] = map[string]interface{}{
			"LowerBound": f.EBSOptimizedBaselineBandwidth.LowerBound.Quantity,
			"UpperBound": f.EBSOptimizedBaselineBandwidth.UpperBound.Quantity,
		}
	}
	if f.EBSOptimizedBaselineThroughput != nil {
		result["EBSOptimizedBaselineThroughput"] = map[string]interface{}{
			"LowerBound": f.EBSOptimizedBaselineThroughput.LowerBound.Quantity,
			"UpperBound": f.EBSOptimizedBaselineThroughput.UpperBound.Quantity,
		}
	}
	if f.EBSOptimizedBaselineIOPS != nil {
		result["EBSOptimizedBaselineIOPS"] = map[string]interface{}{
			"LowerBound": f.EBSOptimizedBaselineIOPS.LowerBound,
			"UpperBound": f.EBSOptimizedBaselineIOPS.UpperBound,
		}
	}
	if f.DedicatedHosts != nil {
		result["DedicatedHosts"] = *f.DedicatedHosts
	}
	if f.Generation != nil {
		result["Generation"] = map[string]interface{}{
			"LowerBound": f.Generation.LowerBound,
			"UpperBound": f.Generation.UpperBound,
		}
	}

	return json.MarshalIndent(result, prefix, indent)
}

// Filters is used to group instance type resource attributes for filtering.
type Filters struct {
	// AvailabilityZones is the AWS Availability Zones where instances will be provisioned.
	// Instance type capacity can vary between availability zones.
	// Will accept zone names or ids
	// Example: us-east-1a, us-east-1b, us-east-2a, etc. OR use1-az1, use2-az2, etc.
	AvailabilityZones *[]string

	// BareMetal is used to only return bare metal instance type results
	BareMetal *bool

	// Burstable is used to only return burstable instance type results like the t* series
	Burstable *bool

	// AutoRecovery is used to filter by instance types that support auto recovery
	AutoRecovery *bool

	// FreeTier is used to filter by instance types that can be used as part of the EC2 free tier
	FreeTier *bool

	// CPUArchitecture of the EC2 instance type
	CPUArchitecture *ec2types.ArchitectureType

	// CPUManufacturer is used to filter instance types with a specific CPU manufacturer
	CPUManufacturer *CPUManufacturer

	// CurrentGeneration returns the latest generation of instance types
	CurrentGeneration *bool

	// EnaSupport returns instances that can support an Elastic Network Adapter.
	EnaSupport *bool

	// EfaSupport returns instances that can support an Elastic Fabric Adapter.
	EfaSupport *bool

	// FPGA is used to only return FPGA instance type results
	Fpga *bool

	// GpusRange filter is a range of acceptable GPU count available to an EC2 instance type
	GpusRange *Int32RangeFilter

	// GpuMemoryRange filter is a range of acceptable GPU memory in Gibibytes (GiB) available to an EC2 instance type in aggreagte across all GPUs.
	GpuMemoryRange *ByteQuantityRangeFilter

	// GPUManufacturer filters by GPU manufacturer
	GPUManufacturer *string

	// GPUModel filter by the GPU model name
	GPUModel *string

	// InferenceAcceleratorsRange filters inference accelerators available to the instance type
	InferenceAcceleratorsRange *IntRangeFilter

	// InferenceAcceleratorManufacturer filters by inference acceleartor manufacturer
	InferenceAcceleratorManufacturer *string

	// InferenceAcceleratorModel filters by inference accelerator model name
	InferenceAcceleratorModel *string

	// HibernationSupported denotes whether EC2 hibernate is supported
	// Possible values are: true or false
	HibernationSupported *bool

	// Hypervisor is used to return only a specific hypervisor backed instance type
	// Possibly values are: xen or nitro
	Hypervisor *ec2types.InstanceTypeHypervisor

	// MaxResults is the maximum number of instance types to return that match the filter criteria
	MaxResults *int

	// MemoryRange filter is a range of acceptable DRAM memory in Gibibytes (GiB) for the instance type
	MemoryRange *ByteQuantityRangeFilter

	// NetworkInterfaces filter is a range of the number of ENI attachments an instance type can support
	NetworkInterfaces *Int32RangeFilter

	// NetworkPerformance filter is a range of network bandwidth an instance type can support
	NetworkPerformance *IntRangeFilter

	// NetworkEncryption filters for instance types that automatically encrypt network traffic in-transit
	NetworkEncryption *bool

	// IPv6 filters for instance types that support IPv6
	IPv6 *bool

	// PlacementGroupStrategy is used to return instance types based on its support
	// for a specific placement group strategy
	// Possible values are: cluster, spread, or partition
	PlacementGroupStrategy *string

	// Region is the AWS Region where instances will be provisioned.
	// Instance type availability can vary between AWS Regions.
	// Example: us-east-1, us-east-2, eu-west-1, etc.
	Region *string

	// RootDeviceType is the backing device of the root storage volume
	// Possible values are: instance-store or ebs
	RootDeviceType *ec2types.RootDeviceType

	// UsageClass of the instance EC2 instance type
	// Possible values are: spot or on-demand
	UsageClass *ec2types.UsageClassType

	// VCpusRange filter is a range of acceptable VCpus for the instance type
	VCpusRange *Int32RangeFilter

	// VcpusToMemoryRatio is a ratio of vcpus to memory expressed as a floating point
	VCpusToMemoryRatio *float64

	// MemoryPerCpuRange is a range filter for memory (in GiB) per vCPU ratio
	// Example: if an instance has 8 GiB memory and 2 vCPUs, the ratio is 4.0 GiB per vCPU
	MemoryPerCpuRange *Float64RangeFilter

	// AllowList is a regex of allowed instance types
	AllowList *regexp.Regexp

	// DenyList is a regex of excluded instance types
	DenyList *regexp.Regexp

	// InstanceTypeBase is a base instance type which is used to retrieve similarly spec'd instance types
	InstanceTypeBase *string

	// Flexible finds an opinionated set of general (c, m, r, t, a, etc.) instance types that match a criteria specified
	// or defaults to 4 vcpus
	Flexible *bool

	// Service filters instance types based on a service's supported list of instance types
	// Example: eks or emr
	Service *string

	// InstanceTypes filters instance types and only allows instance types in this slice
	InstanceTypes *[]string

	// VirtualizationType is used to return instance types that match either hvm or pv virtualization types
	VirtualizationType *ec2types.VirtualizationType

	// PricePerHour is used to return instance types that are equal to or cheaper than the specified price
	PricePerHour *Float64RangeFilter

	// InstanceStorageRange filters on a range of storage available as local disk
	InstanceStorageRange *ByteQuantityRangeFilter

	// NVMEInstanceStorageRange filters on a range of NVMe instance storage only
	// Only includes storage when nvme_support is "required"
	NVMEInstanceStorageRange *ByteQuantityRangeFilter

	// DiskType is the backing storage medium
	// Possible values are: hdd or ssd
	DiskType *string

	// NVME filters for NVME disks, including both EBS and local instance storage
	NVME *bool

	// EBSOptimized filters for instance types that support EBS Optimized
	EBSOptimized *bool

	// DiskEncryption filters for instance types that support EBS Encryption or local storage encryption
	DiskEncryption *bool

	// EBSOptimizedBaselineBandwidth filters on a range of bandwidth that an EBS Optimized volume supports
	EBSOptimizedBaselineBandwidth *ByteQuantityRangeFilter

	// EBSOptimizedBaselineThroughput filters on a range of throughput that an EBS Optimized volume supports
	EBSOptimizedBaselineThroughput *ByteQuantityRangeFilter

	// EBSOptimizedBaselineIOPS filters on a range of IOPS that an EBS Optimized volume supports
	EBSOptimizedBaselineIOPS *IntRangeFilter

	// DedicatedHosts filters on instance types that support dedicated hosts tenancy
	DedicatedHosts *bool

	// Generation filters on the instance type generation
	// i.e. c7i.xlarge is 7
	// NOTE that generation is only comparable per instance family
	// For example, i3 and c5 are both 5th generation, but the Generation filter will
	// only filter on the number in the instance type name.
	Generation *IntRangeFilter
}

type CPUManufacturer string

// Enum values for CPUManufacturer.
const (
	CPUManufacturerAWS   CPUManufacturer = "aws"
	CPUManufacturerAMD   CPUManufacturer = "amd"
	CPUManufacturerIntel CPUManufacturer = "intel"
)

// Values returns all known values for CPUManufacturer. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (CPUManufacturer) Values() []CPUManufacturer {
	return []CPUManufacturer{
		CPUManufacturerAWS,
		CPUManufacturerAMD,
		CPUManufacturerIntel,
	}
}

// ArchitectureTypeAMD64 is a legacy type we support for b/c that isn't in the API.
const (
	ArchitectureTypeAMD64 ec2types.ArchitectureType = "amd64"
)

// ArchitectureTypeAMD64 is a legacy type we support for b/c that isn't in the API.
const (
	VirtualizationTypePv ec2types.VirtualizationType = "pv"
)
