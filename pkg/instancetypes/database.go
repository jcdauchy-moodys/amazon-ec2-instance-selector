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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
)

const (
	DatabaseFileName = "ec2-instance-types.db"
)

// DatabaseProvider implements instance type caching using SQLite database
type DatabaseProvider struct {
	Region          string
	DirectoryPath   string
	FullRefreshTTL  time.Duration
	lastFullRefresh *time.Time
	ec2Client       ec2.DescribeInstanceTypesAPIClient
	db              *sql.DB
	logger          *log.Logger
}

// NewDatabaseProvider creates a new Database Provider for instance type caching
func NewDatabaseProvider(directoryPath string, region string, ttl time.Duration, ec2Client ec2.DescribeInstanceTypesAPIClient) (*DatabaseProvider, error) {
	expandedDirPath, err := homedir.Expand(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("unable to expand database directory path %s: %w", directoryPath, err)
	}

	if err := os.MkdirAll(expandedDirPath, 0755); err != nil {
		return nil, fmt.Errorf("unable to create database directory %s: %w", expandedDirPath, err)
	}

	dbPath := filepath.Join(expandedDirPath, fmt.Sprintf("%s-%s", region, DatabaseFileName))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open database %s: %w", dbPath, err)
	}

	provider := &DatabaseProvider{
		Region:         region,
		DirectoryPath:  expandedDirPath,
		FullRefreshTTL: ttl,
		ec2Client:      ec2Client,
		db:             db,
		logger:         log.New(os.Stderr, "", log.LstdFlags),
	}

	if err := provider.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("unable to initialize database: %w", err)
	}

	// Load last refresh time from database
	if err := provider.loadLastRefreshTime(); err != nil {
		provider.logger.Printf("Warning: could not load last refresh time: %v", err)
	}

	return provider, nil
}

// initializeDatabase creates the necessary tables if they don't exist
func (p *DatabaseProvider) initializeDatabase() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS instance_types (
		instance_type TEXT PRIMARY KEY,
		region TEXT NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS cache_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_instance_types_region ON instance_types(region);
	CREATE INDEX IF NOT EXISTS idx_instance_types_updated_at ON instance_types(updated_at);
	`

	if _, err := p.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// loadLastRefreshTime loads the last full refresh time from the database
func (p *DatabaseProvider) loadLastRefreshTime() error {
	query := "SELECT value FROM cache_metadata WHERE key = ?"
	var timeStr string
	err := p.db.QueryRow(query, "last_full_refresh").Scan(&timeStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No previous refresh time found
		}
		return err
	}

	lastRefresh, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return err
	}

	p.lastFullRefresh = &lastRefresh
	return nil
}

// saveLastRefreshTime saves the last full refresh time to the database
func (p *DatabaseProvider) saveLastRefreshTime() error {
	if p.lastFullRefresh == nil {
		return nil
	}

	query := `
	INSERT OR REPLACE INTO cache_metadata (key, value, updated_at) 
	VALUES (?, ?, CURRENT_TIMESTAMP)
	`
	_, err := p.db.Exec(query, "last_full_refresh", p.lastFullRefresh.Format(time.RFC3339))
	return err
}

// SetLogger sets the logger for the database provider
func (p *DatabaseProvider) SetLogger(logger *log.Logger) {
	p.logger = logger
}

// Get retrieves instance type information from cache or fetches from AWS
func (p *DatabaseProvider) Get(ctx context.Context, instanceTypes []ec2types.InstanceType) ([]*Details, error) {
	p.logger.Printf("Getting instance types from database cache for region %s", p.Region)
	start := time.Now()
	calls := 0
	defer func() {
		p.logger.Printf("Took %s and %d calls to collect Instance Types from database", time.Since(start), calls)
	}()

	instanceTypeDetails := []*Details{}
	var missingInstanceTypes []ec2types.InstanceType

	if len(instanceTypes) != 0 {
		// Check cache for specific instance types
		for _, it := range instanceTypes {
			details, found, err := p.getFromCache(string(it))
			if err != nil {
				p.logger.Printf("Error retrieving %s from cache: %v", it, err)
				missingInstanceTypes = append(missingInstanceTypes, it)
				continue
			}
			if found {
				instanceTypeDetails = append(instanceTypeDetails, details)
			} else {
				missingInstanceTypes = append(missingInstanceTypes, it)
			}
		}

		// If we found all requested types in cache, return early
		if len(missingInstanceTypes) == 0 {
			return instanceTypeDetails, nil
		}
	} else {
		// Check if full refresh is needed
		if p.lastFullRefresh != nil && !p.isFullRefreshNeeded() {
			// Load all from cache
			return p.getAllFromCache()
		}
		// Need full refresh
		missingInstanceTypes = nil // Will fetch all
	}

	// Fetch missing instance types from AWS
	details, err := p.fetchFromAWS(ctx, missingInstanceTypes, calls)
	if err != nil {
		return nil, err
	}

	instanceTypeDetails = append(instanceTypeDetails, details...)

	// If this was a full refresh, update the last refresh time
	if len(instanceTypes) == 0 {
		now := time.Now().UTC()
		p.lastFullRefresh = &now
		if err := p.saveLastRefreshTime(); err != nil {
			p.logger.Printf("Warning: could not save last refresh time: %v", err)
		}
	}

	return instanceTypeDetails, nil
}

// getFromCache retrieves a single instance type from the database cache
func (p *DatabaseProvider) getFromCache(instanceType string) (*Details, bool, error) {
	query := "SELECT data FROM instance_types WHERE instance_type = ? AND region = ?"
	var dataJSON string
	err := p.db.QueryRow(query, instanceType, p.Region).Scan(&dataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	var details Details
	if err := json.Unmarshal([]byte(dataJSON), &details); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal instance type data: %w", err)
	}

	return &details, true, nil
}

// getAllFromCache retrieves all instance types from the database cache
func (p *DatabaseProvider) getAllFromCache() ([]*Details, error) {
	query := "SELECT data FROM instance_types WHERE region = ?"
	rows, err := p.db.Query(query, p.Region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instanceTypeDetails []*Details
	for rows.Next() {
		var dataJSON string
		if err := rows.Scan(&dataJSON); err != nil {
			return nil, err
		}

		var details Details
		if err := json.Unmarshal([]byte(dataJSON), &details); err != nil {
			p.logger.Printf("Warning: failed to unmarshal cached instance type data: %v", err)
			continue
		}

		instanceTypeDetails = append(instanceTypeDetails, &details)
	}

	return instanceTypeDetails, rows.Err()
}

// fetchFromAWS fetches instance types from AWS API and stores them in cache
func (p *DatabaseProvider) fetchFromAWS(ctx context.Context, instanceTypes []ec2types.InstanceType, calls int) ([]*Details, error) {
	describeInstanceTypeOpts := &ec2.DescribeInstanceTypesInput{}
	if len(instanceTypes) > 0 {
		describeInstanceTypeOpts.InstanceTypes = instanceTypes
	}

	var instanceTypeDetails []*Details
	s := ec2.NewDescribeInstanceTypesPaginator(p.ec2Client, describeInstanceTypeOpts)

	for s.HasMorePages() {
		calls++
		instanceTypeOutput, err := s.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get next instance types page: %w", err)
		}

		for _, instanceTypeInfo := range instanceTypeOutput.InstanceTypes {
			itDetails := &Details{InstanceTypeInfo: instanceTypeInfo}
			instanceTypeDetails = append(instanceTypeDetails, itDetails)

			// Store in database cache
			if err := p.storeInCache(string(instanceTypeInfo.InstanceType), itDetails); err != nil {
				p.logger.Printf("Warning: failed to store %s in cache: %v", instanceTypeInfo.InstanceType, err)
			}
		}
	}

	return instanceTypeDetails, nil
}

// storeInCache stores an instance type in the database cache
func (p *DatabaseProvider) storeInCache(instanceType string, details *Details) error {
	dataJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal instance type data: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO instance_types (instance_type, region, data, updated_at) 
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err = p.db.Exec(query, instanceType, p.Region, string(dataJSON))
	return err
}

// isFullRefreshNeeded checks if a full refresh is needed based on TTL
func (p *DatabaseProvider) isFullRefreshNeeded() bool {
	if p.FullRefreshTTL <= 0 {
		return true
	}
	if p.lastFullRefresh == nil {
		return true
	}
	return time.Since(*p.lastFullRefresh) > p.FullRefreshTTL
}

// Save is a no-op for database provider as data is saved immediately
func (p *DatabaseProvider) Save() error {
	return p.saveLastRefreshTime()
}

// Clear removes all cached instance types for the current region
func (p *DatabaseProvider) Clear() error {
	query := "DELETE FROM instance_types WHERE region = ?"
	_, err := p.db.Exec(query, p.Region)
	if err != nil {
		return fmt.Errorf("failed to clear cache for region %s: %w", p.Region, err)
	}

	// Also clear metadata
	metaQuery := "DELETE FROM cache_metadata WHERE key = ?"
	_, err = p.db.Exec(metaQuery, "last_full_refresh")
	if err != nil {
		p.logger.Printf("Warning: failed to clear cache metadata: %v", err)
	}

	return nil
}

// CacheCount returns the number of cached instance types for the current region
func (p *DatabaseProvider) CacheCount() int {
	query := "SELECT COUNT(*) FROM instance_types WHERE region = ?"
	var count int
	err := p.db.QueryRow(query, p.Region).Scan(&count)
	if err != nil {
		p.logger.Printf("Warning: failed to get cache count: %v", err)
		return 0
	}
	return count
}

// Close closes the database connection
func (p *DatabaseProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
