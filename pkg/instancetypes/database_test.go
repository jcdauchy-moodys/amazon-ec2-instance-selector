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

package instancetypes

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2Client struct{}

func (m mockEC2Client) DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	// Return a simple mock response for testing
	output := &ec2.DescribeInstanceTypesOutput{
		InstanceTypes: []ec2types.InstanceTypeInfo{
			{
				InstanceType: "t2.micro",
				VCpuInfo: &ec2types.VCpuInfo{
					DefaultVCpus: aws.Int32(1),
				},
				MemoryInfo: &ec2types.MemoryInfo{
					SizeInMiB: aws.Int64(1024),
				},
				CurrentGeneration: aws.Bool(true),
				Hypervisor:        "xen",
			},
		},
	}
	return output, nil
}

func TestDatabaseProvider_NewAndBasicOperations(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "ec2-instance-selector-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a database provider
	provider, err := NewDatabaseProvider(tmpDir, "us-east-1", time.Hour, &mockEC2Client{})
	if err != nil {
		t.Fatalf("Failed to create database provider: %v", err)
	}
	defer provider.Close()

	// Test that cache count is initially 0
	if count := provider.CacheCount(); count != 0 {
		t.Errorf("Expected cache count to be 0, got %d", count)
	}

	// Test getting instance types (should trigger API call and caching)
	ctx := context.Background()
	instanceTypes, err := provider.Get(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get instance types: %v", err)
	}

	if len(instanceTypes) != 1 {
		t.Errorf("Expected 1 instance type, got %d", len(instanceTypes))
	}

	if instanceTypes[0].InstanceType != "t2.micro" {
		t.Errorf("Expected instance type t2.micro, got %s", instanceTypes[0].InstanceType)
	}

	// Test that cache count is now 1
	if count := provider.CacheCount(); count != 1 {
		t.Errorf("Expected cache count to be 1, got %d", count)
	}

	// Test getting specific instance type from cache
	specificTypes := []ec2types.InstanceType{"t2.micro"}
	cachedTypes, err := provider.Get(ctx, specificTypes)
	if err != nil {
		t.Fatalf("Failed to get specific instance types: %v", err)
	}

	if len(cachedTypes) != 1 {
		t.Errorf("Expected 1 cached instance type, got %d", len(cachedTypes))
	}

	// Test clearing the cache
	if err := provider.Clear(); err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Test that cache count is back to 0
	if count := provider.CacheCount(); count != 0 {
		t.Errorf("Expected cache count to be 0 after clear, got %d", count)
	}
}

func TestDatabaseProvider_DatabaseFileCreation(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "ec2-instance-selector-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	region := "us-west-2"
	provider, err := NewDatabaseProvider(tmpDir, region, time.Hour, &mockEC2Client{})
	if err != nil {
		t.Fatalf("Failed to create database provider: %v", err)
	}
	defer provider.Close()

	// Check that the database file was created
	expectedPath := filepath.Join(tmpDir, region+"-"+DatabaseFileName)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected database file to be created at %s", expectedPath)
	}
}

func TestLoadProviderFromOrNew_DatabasePreferred(t *testing.T) {
	// Create temporary directories
	cacheDir, err := os.MkdirTemp("", "ec2-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create cache temp directory: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	dbDir, err := os.MkdirTemp("", "ec2-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create db temp directory: %v", err)
	}
	defer os.RemoveAll(dbDir)

	// Test that database provider is created when dbDir is provided
	provider, err := LoadProviderFromOrNew(cacheDir, dbDir, "us-east-1", time.Hour, &mockEC2Client{})
	if err != nil {
		t.Fatalf("Failed to load provider: %v", err)
	}

	// Check that it's a database provider by type assertion
	if dbProvider, ok := provider.(*DatabaseProvider); ok {
		defer dbProvider.Close()
		if dbProvider.Region != "us-east-1" {
			t.Errorf("Expected region us-east-1, got %s", dbProvider.Region)
		}
	} else {
		t.Error("Expected DatabaseProvider when dbDir is provided")
	}
}

func TestLoadProviderFromOrNew_FileProviderFallback(t *testing.T) {
	// Create temporary directory for cache
	cacheDir, err := os.MkdirTemp("", "ec2-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create cache temp directory: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	// Test that file provider is created when dbDir is empty
	provider, err := LoadProviderFromOrNew(cacheDir, "", "us-east-1", time.Hour, &mockEC2Client{})
	if err != nil {
		t.Fatalf("Failed to load provider: %v", err)
	}

	// Check that it's a file provider by type assertion
	if _, ok := provider.(*Provider); !ok {
		t.Error("Expected Provider (file-based) when dbDir is empty")
	}
}
