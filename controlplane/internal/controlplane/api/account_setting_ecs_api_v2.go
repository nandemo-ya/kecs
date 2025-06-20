package api

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// PutAccountSettingV2 implements the PutAccountSetting operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) PutAccountSettingV2(ctx context.Context, req *ecs.PutAccountSettingInput) (*ecs.PutAccountSettingOutput, error) {
	// TODO: Implement PutAccountSetting
	return nil, fmt.Errorf("PutAccountSetting not implemented")
}

// PutAccountSettingDefaultV2 implements the PutAccountSettingDefault operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) PutAccountSettingDefaultV2(ctx context.Context, req *ecs.PutAccountSettingDefaultInput) (*ecs.PutAccountSettingDefaultOutput, error) {
	// TODO: Implement PutAccountSettingDefault
	return nil, fmt.Errorf("PutAccountSettingDefault not implemented")
}

// DeleteAccountSettingV2 implements the DeleteAccountSetting operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DeleteAccountSettingV2(ctx context.Context, req *ecs.DeleteAccountSettingInput) (*ecs.DeleteAccountSettingOutput, error) {
	// TODO: Implement DeleteAccountSetting
	return nil, fmt.Errorf("DeleteAccountSetting not implemented")
}

// ListAccountSettingsV2 implements the ListAccountSettings operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) ListAccountSettingsV2(ctx context.Context, req *ecs.ListAccountSettingsInput) (*ecs.ListAccountSettingsOutput, error) {
	// TODO: Implement ListAccountSettings
	return nil, fmt.Errorf("ListAccountSettings not implemented")
}