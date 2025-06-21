package api

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// Converter functions from AWS SDK v2 types to generated types

// ConvertToGeneratedListClustersRequest converts AWS SDK ListClustersInput to generated type
func ConvertToGeneratedListClustersRequest(input *ecs.ListClustersInput) *generated.ListClustersRequest {
	if input == nil {
		return &generated.ListClustersRequest{}
	}
	
	req := &generated.ListClustersRequest{}
	if input.MaxResults != nil {
		req.MaxResults = input.MaxResults
	}
	if input.NextToken != nil {
		req.NextToken = input.NextToken
	}
	return req
}

// ConvertFromGeneratedListClustersResponse converts generated response to AWS SDK type
func ConvertFromGeneratedListClustersResponse(resp *generated.ListClustersResponse) *ecs.ListClustersOutput {
	if resp == nil {
		return &ecs.ListClustersOutput{}
	}
	
	output := &ecs.ListClustersOutput{
		ClusterArns: resp.ClusterArns,
	}
	if resp.NextToken != nil {
		output.NextToken = resp.NextToken
	}
	return output
}

// ConvertToGeneratedCreateClusterRequest converts AWS SDK CreateClusterInput to generated type
func ConvertToGeneratedCreateClusterRequest(input *ecs.CreateClusterInput) *generated.CreateClusterRequest {
	if input == nil {
		return &generated.CreateClusterRequest{}
	}
	
	req := &generated.CreateClusterRequest{
		ClusterName: input.ClusterName,
	}
	
	// Convert settings
	if len(input.Settings) > 0 {
		req.Settings = make([]generated.ClusterSetting, 0, len(input.Settings))
		for _, s := range input.Settings {
			setting := generated.ClusterSetting{
				Name: (*generated.ClusterSettingName)(aws.String(string(s.Name))),
			}
			if s.Value != nil {
				setting.Value = s.Value
			}
			req.Settings = append(req.Settings, setting)
		}
	}
	
	// Convert tags
	if len(input.Tags) > 0 {
		req.Tags = ConvertToGeneratedTags(input.Tags)
	}
	
	// Convert configuration
	if input.Configuration != nil {
		req.Configuration = ConvertToGeneratedClusterConfiguration(input.Configuration)
	}
	
	// Convert capacity providers
	req.CapacityProviders = input.CapacityProviders
	
	// Convert default capacity provider strategy
	if len(input.DefaultCapacityProviderStrategy) > 0 {
		req.DefaultCapacityProviderStrategy = ConvertToGeneratedCapacityProviderStrategy(input.DefaultCapacityProviderStrategy)
	}
	
	// Convert service connect defaults
	if input.ServiceConnectDefaults != nil {
		req.ServiceConnectDefaults = ConvertToGeneratedServiceConnectDefaults(input.ServiceConnectDefaults)
	}
	
	return req
}

// ConvertFromGeneratedCreateClusterResponse converts generated response to AWS SDK type
func ConvertFromGeneratedCreateClusterResponse(resp *generated.CreateClusterResponse) *ecs.CreateClusterOutput {
	if resp == nil || resp.Cluster == nil {
		return &ecs.CreateClusterOutput{}
	}
	
	return &ecs.CreateClusterOutput{
		Cluster: ConvertFromGeneratedCluster(resp.Cluster),
	}
}

// ConvertToGeneratedTags converts AWS SDK tags to generated tags
func ConvertToGeneratedTags(tags []ecstypes.Tag) []generated.Tag {
	if len(tags) == 0 {
		return nil
	}
	
	result := make([]generated.Tag, 0, len(tags))
	for _, tag := range tags {
		var genTag generated.Tag
		if tag.Key != nil {
			key := generated.TagKey(*tag.Key)
			genTag.Key = &key
		}
		if tag.Value != nil {
			value := generated.TagValue(*tag.Value)
			genTag.Value = &value
		}
		result = append(result, genTag)
	}
	return result
}

// ConvertFromGeneratedTags converts generated tags to AWS SDK tags
func ConvertFromGeneratedTags(tags []generated.Tag) []ecstypes.Tag {
	if len(tags) == 0 {
		return nil
	}
	
	result := make([]ecstypes.Tag, 0, len(tags))
	for _, tag := range tags {
		var sdkTag ecstypes.Tag
		if tag.Key != nil {
			key := string(*tag.Key)
			sdkTag.Key = &key
		}
		if tag.Value != nil {
			value := string(*tag.Value)
			sdkTag.Value = &value
		}
		result = append(result, sdkTag)
	}
	return result
}

// ConvertToGeneratedClusterConfiguration converts AWS SDK cluster configuration to generated type
func ConvertToGeneratedClusterConfiguration(config *ecstypes.ClusterConfiguration) *generated.ClusterConfiguration {
	if config == nil {
		return nil
	}
	
	result := &generated.ClusterConfiguration{}
	
	if config.ExecuteCommandConfiguration != nil {
		result.ExecuteCommandConfiguration = &generated.ExecuteCommandConfiguration{
			KmsKeyId: config.ExecuteCommandConfiguration.KmsKeyId,
		}
		if config.ExecuteCommandConfiguration.LogConfiguration != nil {
			result.ExecuteCommandConfiguration.LogConfiguration = &generated.ExecuteCommandLogConfiguration{
				CloudWatchEncryptionEnabled: &config.ExecuteCommandConfiguration.LogConfiguration.CloudWatchEncryptionEnabled,
				CloudWatchLogGroupName:      config.ExecuteCommandConfiguration.LogConfiguration.CloudWatchLogGroupName,
				S3BucketName:                config.ExecuteCommandConfiguration.LogConfiguration.S3BucketName,
				S3EncryptionEnabled:         &config.ExecuteCommandConfiguration.LogConfiguration.S3EncryptionEnabled,
				S3KeyPrefix:                 config.ExecuteCommandConfiguration.LogConfiguration.S3KeyPrefix,
			}
		}
		if config.ExecuteCommandConfiguration.Logging != "" {
			logging := generated.ExecuteCommandLogging(config.ExecuteCommandConfiguration.Logging)
			result.ExecuteCommandConfiguration.Logging = &logging
		}
	}
	
	if config.ManagedStorageConfiguration != nil {
		result.ManagedStorageConfiguration = &generated.ManagedStorageConfiguration{
			FargateEphemeralStorageKmsKeyId: config.ManagedStorageConfiguration.FargateEphemeralStorageKmsKeyId,
			KmsKeyId:                        config.ManagedStorageConfiguration.KmsKeyId,
		}
	}
	
	return result
}

// ConvertToGeneratedCapacityProviderStrategy converts AWS SDK capacity provider strategy to generated type
func ConvertToGeneratedCapacityProviderStrategy(strategy []ecstypes.CapacityProviderStrategyItem) []generated.CapacityProviderStrategyItem {
	if len(strategy) == 0 {
		return nil
	}
	
	result := make([]generated.CapacityProviderStrategyItem, 0, len(strategy))
	for _, item := range strategy {
		genItem := generated.CapacityProviderStrategyItem{
			CapacityProvider: item.CapacityProvider,
		}
		if item.Base != 0 {
			base := generated.CapacityProviderStrategyItemBase(item.Base)
			genItem.Base = &base
		}
		if item.Weight != 0 {
			weight := generated.CapacityProviderStrategyItemWeight(item.Weight)
			genItem.Weight = &weight
		}
		result = append(result, genItem)
	}
	return result
}

// ConvertToGeneratedServiceConnectDefaults converts AWS SDK service connect defaults to generated type
func ConvertToGeneratedServiceConnectDefaults(defaults *ecstypes.ClusterServiceConnectDefaultsRequest) *generated.ClusterServiceConnectDefaultsRequest {
	if defaults == nil {
		return nil
	}
	
	return &generated.ClusterServiceConnectDefaultsRequest{
		Namespace: defaults.Namespace,
	}
}

// ConvertFromGeneratedCluster converts generated cluster to AWS SDK type
func ConvertFromGeneratedCluster(cluster *generated.Cluster) *ecstypes.Cluster {
	if cluster == nil {
		return nil
	}
	
	result := &ecstypes.Cluster{
		ClusterArn:                        cluster.ClusterArn,
		ClusterName:                       cluster.ClusterName,
		Status:                            cluster.Status,
		ActiveServicesCount:               GetInt32Value(cluster.ActiveServicesCount),
		RunningTasksCount:                 GetInt32Value(cluster.RunningTasksCount),
		PendingTasksCount:                 GetInt32Value(cluster.PendingTasksCount),
		RegisteredContainerInstancesCount: GetInt32Value(cluster.RegisteredContainerInstancesCount),
	}
	
	// Convert settings
	if len(cluster.Settings) > 0 {
		result.Settings = make([]ecstypes.ClusterSetting, 0, len(cluster.Settings))
		for _, s := range cluster.Settings {
			if s.Name != nil {
				setting := ecstypes.ClusterSetting{
					Name:  ecstypes.ClusterSettingName(string(*s.Name)),
					Value: s.Value,
				}
				result.Settings = append(result.Settings, setting)
			}
		}
	}
	
	// Convert tags
	if len(cluster.Tags) > 0 {
		result.Tags = ConvertFromGeneratedTags(cluster.Tags)
	}
	
	// Convert configuration
	if cluster.Configuration != nil {
		result.Configuration = ConvertFromGeneratedClusterConfiguration(cluster.Configuration)
	}
	
	// Convert attachments
	if len(cluster.Attachments) > 0 {
		result.Attachments = ConvertFromGeneratedAttachments(cluster.Attachments)
	}
	
	result.AttachmentsStatus = cluster.AttachmentsStatus
	result.CapacityProviders = cluster.CapacityProviders
	
	// Convert default capacity provider strategy
	if len(cluster.DefaultCapacityProviderStrategy) > 0 {
		result.DefaultCapacityProviderStrategy = ConvertFromGeneratedCapacityProviderStrategy(cluster.DefaultCapacityProviderStrategy)
	}
	
	// Convert service connect defaults
	if cluster.ServiceConnectDefaults != nil {
		result.ServiceConnectDefaults = &ecstypes.ClusterServiceConnectDefaults{
			Namespace: cluster.ServiceConnectDefaults.Namespace,
		}
	}
	
	// Convert statistics
	if len(cluster.Statistics) > 0 {
		result.Statistics = make([]ecstypes.KeyValuePair, 0, len(cluster.Statistics))
		for _, kv := range cluster.Statistics {
			result.Statistics = append(result.Statistics, ecstypes.KeyValuePair{
				Name:  kv.Name,
				Value: kv.Value,
			})
		}
	}
	
	return result
}

// ConvertFromGeneratedClusterConfiguration converts generated cluster configuration to AWS SDK type
func ConvertFromGeneratedClusterConfiguration(config *generated.ClusterConfiguration) *ecstypes.ClusterConfiguration {
	if config == nil {
		return nil
	}
	
	result := &ecstypes.ClusterConfiguration{}
	
	if config.ExecuteCommandConfiguration != nil {
		result.ExecuteCommandConfiguration = &ecstypes.ExecuteCommandConfiguration{
			KmsKeyId: config.ExecuteCommandConfiguration.KmsKeyId,
		}
		if config.ExecuteCommandConfiguration.LogConfiguration != nil {
			logConfig := &ecstypes.ExecuteCommandLogConfiguration{
				CloudWatchLogGroupName: config.ExecuteCommandConfiguration.LogConfiguration.CloudWatchLogGroupName,
				S3BucketName:          config.ExecuteCommandConfiguration.LogConfiguration.S3BucketName,
				S3KeyPrefix:           config.ExecuteCommandConfiguration.LogConfiguration.S3KeyPrefix,
			}
			// Convert bool pointers to bool values
			if config.ExecuteCommandConfiguration.LogConfiguration.CloudWatchEncryptionEnabled != nil {
				logConfig.CloudWatchEncryptionEnabled = *config.ExecuteCommandConfiguration.LogConfiguration.CloudWatchEncryptionEnabled
			}
			if config.ExecuteCommandConfiguration.LogConfiguration.S3EncryptionEnabled != nil {
				logConfig.S3EncryptionEnabled = *config.ExecuteCommandConfiguration.LogConfiguration.S3EncryptionEnabled
			}
			result.ExecuteCommandConfiguration.LogConfiguration = logConfig
		}
		if config.ExecuteCommandConfiguration.Logging != nil {
			result.ExecuteCommandConfiguration.Logging = ecstypes.ExecuteCommandLogging(*config.ExecuteCommandConfiguration.Logging)
		}
	}
	
	if config.ManagedStorageConfiguration != nil {
		result.ManagedStorageConfiguration = &ecstypes.ManagedStorageConfiguration{
			FargateEphemeralStorageKmsKeyId: config.ManagedStorageConfiguration.FargateEphemeralStorageKmsKeyId,
			KmsKeyId:                        config.ManagedStorageConfiguration.KmsKeyId,
		}
	}
	
	return result
}

// ConvertFromGeneratedAttachments converts generated attachments to AWS SDK type
func ConvertFromGeneratedAttachments(attachments []generated.Attachment) []ecstypes.Attachment {
	if len(attachments) == 0 {
		return nil
	}
	
	result := make([]ecstypes.Attachment, 0, len(attachments))
	for _, a := range attachments {
		attachment := ecstypes.Attachment{
			Id:     a.Id,
			Status: a.Status,
			Type:   a.Type,
		}
		
		// Convert details
		if len(a.Details) > 0 {
			attachment.Details = make([]ecstypes.KeyValuePair, 0, len(a.Details))
			for _, kv := range a.Details {
				attachment.Details = append(attachment.Details, ecstypes.KeyValuePair{
					Name:  kv.Name,
					Value: kv.Value,
				})
			}
		}
		
		result = append(result, attachment)
	}
	return result
}

// ConvertFromGeneratedCapacityProviderStrategy converts generated capacity provider strategy to AWS SDK type
func ConvertFromGeneratedCapacityProviderStrategy(strategy []generated.CapacityProviderStrategyItem) []ecstypes.CapacityProviderStrategyItem {
	if len(strategy) == 0 {
		return nil
	}
	
	result := make([]ecstypes.CapacityProviderStrategyItem, 0, len(strategy))
	for _, item := range strategy {
		sdkItem := ecstypes.CapacityProviderStrategyItem{
			CapacityProvider: item.CapacityProvider,
		}
		if item.Base != nil {
			sdkItem.Base = int32(*item.Base)
		}
		if item.Weight != nil {
			sdkItem.Weight = int32(*item.Weight)
		}
		result = append(result, sdkItem)
	}
	return result
}

// Helper function to get int32 value safely
func GetInt32Value(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// Convert DescribeClusters types
func ConvertToGeneratedDescribeClustersRequest(input *ecs.DescribeClustersInput) *generated.DescribeClustersRequest {
	if input == nil {
		return &generated.DescribeClustersRequest{}
	}
	
	req := &generated.DescribeClustersRequest{
		Clusters: input.Clusters,
	}
	
	// Convert include fields
	if len(input.Include) > 0 {
		req.Include = make([]generated.ClusterField, 0, len(input.Include))
		for _, field := range input.Include {
			req.Include = append(req.Include, generated.ClusterField(field))
		}
	}
	
	return req
}

func ConvertFromGeneratedDescribeClustersResponse(resp *generated.DescribeClustersResponse) *ecs.DescribeClustersOutput {
	if resp == nil {
		return &ecs.DescribeClustersOutput{}
	}
	
	output := &ecs.DescribeClustersOutput{}
	
	// Convert clusters
	if len(resp.Clusters) > 0 {
		output.Clusters = make([]ecstypes.Cluster, 0, len(resp.Clusters))
		for _, cluster := range resp.Clusters {
			output.Clusters = append(output.Clusters, *ConvertFromGeneratedCluster(&cluster))
		}
	}
	
	// Convert failures
	if len(resp.Failures) > 0 {
		output.Failures = make([]ecstypes.Failure, 0, len(resp.Failures))
		for _, failure := range resp.Failures {
			output.Failures = append(output.Failures, ecstypes.Failure{
				Arn:    failure.Arn,
				Detail: failure.Detail,
				Reason: failure.Reason,
			})
		}
	}
	
	return output
}

// Convert DeleteCluster types
func ConvertToGeneratedDeleteClusterRequest(input *ecs.DeleteClusterInput) *generated.DeleteClusterRequest {
	if input == nil {
		return &generated.DeleteClusterRequest{}
	}
	
	return &generated.DeleteClusterRequest{
		Cluster: input.Cluster,
	}
}

func ConvertFromGeneratedDeleteClusterResponse(resp *generated.DeleteClusterResponse) *ecs.DeleteClusterOutput {
	if resp == nil || resp.Cluster == nil {
		return &ecs.DeleteClusterOutput{}
	}
	
	return &ecs.DeleteClusterOutput{
		Cluster: ConvertFromGeneratedCluster(resp.Cluster),
	}
}

// Convert UpdateCluster types
func ConvertToGeneratedUpdateClusterRequest(input *ecs.UpdateClusterInput) *generated.UpdateClusterRequest {
	if input == nil {
		return &generated.UpdateClusterRequest{}
	}
	
	req := &generated.UpdateClusterRequest{
		Cluster: input.Cluster,
	}
	
	// Convert settings
	if len(input.Settings) > 0 {
		req.Settings = make([]generated.ClusterSetting, 0, len(input.Settings))
		for _, s := range input.Settings {
			setting := generated.ClusterSetting{
				Name: (*generated.ClusterSettingName)(aws.String(string(s.Name))),
			}
			if s.Value != nil {
				setting.Value = s.Value
			}
			req.Settings = append(req.Settings, setting)
		}
	}
	
	// Convert configuration
	if input.Configuration != nil {
		req.Configuration = ConvertToGeneratedClusterConfiguration(input.Configuration)
	}
	
	// Convert service connect defaults
	if input.ServiceConnectDefaults != nil {
		req.ServiceConnectDefaults = ConvertToGeneratedServiceConnectDefaults(input.ServiceConnectDefaults)
	}
	
	return req
}

func ConvertFromGeneratedUpdateClusterResponse(resp *generated.UpdateClusterResponse) *ecs.UpdateClusterOutput {
	if resp == nil || resp.Cluster == nil {
		return &ecs.UpdateClusterOutput{}
	}
	
	return &ecs.UpdateClusterOutput{
		Cluster: ConvertFromGeneratedCluster(resp.Cluster),
	}
}