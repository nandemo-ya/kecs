package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// PutAccountSetting implements the PutAccountSetting operation
func (api *DefaultECSAPI) PutAccountSetting(ctx context.Context, req *generated.PutAccountSettingRequest) (*generated.PutAccountSettingResponse, error) {
	// TODO: Implement PutAccountSetting
	return nil, fmt.Errorf("PutAccountSetting not implemented")
}

// PutAccountSettingDefault implements the PutAccountSettingDefault operation
func (api *DefaultECSAPI) PutAccountSettingDefault(ctx context.Context, req *generated.PutAccountSettingDefaultRequest) (*generated.PutAccountSettingDefaultResponse, error) {
	// TODO: Implement PutAccountSettingDefault
	return nil, fmt.Errorf("PutAccountSettingDefault not implemented")
}

// DeleteAccountSetting implements the DeleteAccountSetting operation
func (api *DefaultECSAPI) DeleteAccountSetting(ctx context.Context, req *generated.DeleteAccountSettingRequest) (*generated.DeleteAccountSettingResponse, error) {
	// TODO: Implement DeleteAccountSetting
	return nil, fmt.Errorf("DeleteAccountSetting not implemented")
}

// ListAccountSettings implements the ListAccountSettings operation
func (api *DefaultECSAPI) ListAccountSettings(ctx context.Context, req *generated.ListAccountSettingsRequest) (*generated.ListAccountSettingsResponse, error) {
	// TODO: Implement ListAccountSettings
	return nil, fmt.Errorf("ListAccountSettings not implemented")
}