package route53

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// Client wraps AWS Route53 client for LocalStack integration
type Client struct {
	client          *route53.Client
	localstackMode  bool
	defaultTTL      int64
	changeIDCounter int
}

// NewClient creates a new Route53 client
func NewClient(ctx context.Context, endpoint string) (*Client, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		// LocalStack mode
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					if service == route53.ServiceID {
						return aws.Endpoint{
							URL:               endpoint,
							HostnameImmutable: true,
						}, nil
					}
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				})),
			config.WithRegion("us-east-1"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load LocalStack config: %w", err)
		}
	} else {
		// Standard AWS mode
		cfg, err = config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	return &Client{
		client:         route53.NewFromConfig(cfg),
		localstackMode: endpoint != "",
		defaultTTL:     60,
	}, nil
}

// CreateHostedZone creates a new hosted zone
func (c *Client) CreateHostedZone(ctx context.Context, name string, vpc *VPCConfig) (*HostedZone, error) {
	// Ensure name ends with a dot
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	input := &route53.CreateHostedZoneInput{
		Name:            aws.String(name),
		CallerReference: aws.String(fmt.Sprintf("kecs-%s-%d", name, c.changeIDCounter)),
		HostedZoneConfig: &types.HostedZoneConfig{
			Comment:     aws.String(fmt.Sprintf("KECS Service Discovery zone for %s", name)),
			PrivateZone: true,
		},
	}

	// Add VPC configuration if provided
	if vpc != nil {
		input.VPC = &types.VPC{
			VPCId:     aws.String(vpc.VPCID),
			VPCRegion: types.VPCRegion(vpc.Region),
		}
	}

	output, err := c.client.CreateHostedZone(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create hosted zone: %w", err)
	}

	c.changeIDCounter++

	return &HostedZone{
		ID:   aws.ToString(output.HostedZone.Id),
		Name: aws.ToString(output.HostedZone.Name),
	}, nil
}

// GetHostedZone retrieves a hosted zone by ID
func (c *Client) GetHostedZone(ctx context.Context, zoneID string) (*HostedZone, error) {
	output, err := c.client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: aws.String(zoneID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get hosted zone: %w", err)
	}

	return &HostedZone{
		ID:   aws.ToString(output.HostedZone.Id),
		Name: aws.ToString(output.HostedZone.Name),
	}, nil
}

// ListHostedZones lists all hosted zones
func (c *Client) ListHostedZones(ctx context.Context) ([]*HostedZone, error) {
	output, err := c.client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list hosted zones: %w", err)
	}

	zones := make([]*HostedZone, 0, len(output.HostedZones))
	for _, zone := range output.HostedZones {
		zones = append(zones, &HostedZone{
			ID:   aws.ToString(zone.Id),
			Name: aws.ToString(zone.Name),
		})
	}

	return zones, nil
}

// DeleteHostedZone deletes a hosted zone
func (c *Client) DeleteHostedZone(ctx context.Context, zoneID string) error {
	_, err := c.client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
		Id: aws.String(zoneID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete hosted zone: %w", err)
	}
	return nil
}

// UpsertARecord creates or updates an A record
func (c *Client) UpsertARecord(ctx context.Context, zoneID, name string, ips []string) error {
	if len(ips) == 0 {
		logging.Debug("No IPs provided for A record, skipping", "name", name)
		return nil
	}

	// Build resource records
	resourceRecords := make([]types.ResourceRecord, 0, len(ips))
	for _, ip := range ips {
		resourceRecords = append(resourceRecords, types.ResourceRecord{
			Value: aws.String(ip),
		})
	}

	change := &types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name:            aws.String(name),
			Type:            types.RRTypeA,
			TTL:             aws.Int64(c.defaultTTL),
			ResourceRecords: resourceRecords,
		},
	}

	return c.changeResourceRecordSets(ctx, zoneID, []*types.Change{change})
}

// UpsertSRVRecord creates or updates an SRV record
func (c *Client) UpsertSRVRecord(ctx context.Context, zoneID, name string, targets []SRVTarget) error {
	if len(targets) == 0 {
		logging.Debug("No targets provided for SRV record, skipping", "name", name)
		return nil
	}

	// Build resource records
	resourceRecords := make([]types.ResourceRecord, 0, len(targets))
	for _, target := range targets {
		// SRV record format: priority weight port target
		value := fmt.Sprintf("%d %d %d %s", target.Priority, target.Weight, target.Port, target.Target)
		resourceRecords = append(resourceRecords, types.ResourceRecord{
			Value: aws.String(value),
		})
	}

	change := &types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name:            aws.String(name),
			Type:            types.RRTypeSrv,
			TTL:             aws.Int64(c.defaultTTL),
			ResourceRecords: resourceRecords,
		},
	}

	return c.changeResourceRecordSets(ctx, zoneID, []*types.Change{change})
}

// DeleteRecord deletes a resource record
func (c *Client) DeleteRecord(ctx context.Context, zoneID, name string, recordType types.RRType) error {
	// First, get the existing record to delete it properly
	records, err := c.ListResourceRecordSets(ctx, zoneID, name)
	if err != nil {
		return fmt.Errorf("failed to list records for deletion: %w", err)
	}

	var recordToDelete *types.ResourceRecordSet
	for _, record := range records {
		if record.Type == recordType && aws.ToString(record.Name) == name {
			recordToDelete = &record
			break
		}
	}

	if recordToDelete == nil {
		logging.Debug("Record not found for deletion", "name", name, "type", recordType)
		return nil
	}

	change := &types.Change{
		Action:            types.ChangeActionDelete,
		ResourceRecordSet: recordToDelete,
	}

	return c.changeResourceRecordSets(ctx, zoneID, []*types.Change{change})
}

// ListResourceRecordSets lists resource record sets in a hosted zone
func (c *Client) ListResourceRecordSets(ctx context.Context, zoneID, name string) ([]types.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	if name != "" {
		input.StartRecordName = aws.String(name)
	}

	output, err := c.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource record sets: %w", err)
	}

	return output.ResourceRecordSets, nil
}

// changeResourceRecordSets applies changes to resource record sets
func (c *Client) changeResourceRecordSets(ctx context.Context, zoneID string, changes []*types.Change) error {
	if len(changes) == 0 {
		return nil
	}

	// Convert []*types.Change to []types.Change
	changeSlice := make([]types.Change, 0, len(changes))
	for _, change := range changes {
		if change != nil {
			changeSlice = append(changeSlice, *change)
		}
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String("KECS Service Discovery update"),
			Changes: changeSlice,
		},
	}

	_, err := c.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to change resource record sets: %w", err)
	}

	logging.Debug("Successfully updated Route53 records", "zoneID", zoneID, "changes", len(changes))
	return nil
}

// Types

// HostedZone represents a Route53 hosted zone
type HostedZone struct {
	ID   string
	Name string
}

// VPCConfig represents VPC configuration for private hosted zones
type VPCConfig struct {
	VPCID  string
	Region string
}

// SRVTarget represents a target for an SRV record
type SRVTarget struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
}
