package postgres_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("AccountSettingStore", func() {
	var (
		store storage.Storage
		ctx   context.Context
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Create", func() {
		Context("when creating a new account setting", func() {
			It("should create the setting successfully", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/test-user",
					Name:         "serviceLongArnFormat",
					Value:        "enabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}

				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				// Verify setting was created
				retrieved, err := store.AccountSettingStore().Get(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Value).To(Equal(setting.Value))
			})
		})

		Context("when creating a duplicate account setting", func() {
			It("should return ErrResourceAlreadyExists", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/duplicate",
					Name:         "taskLongArnFormat",
					Value:        "enabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}

				// Create first setting
				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				// Try to upsert duplicate - it should update not error
				setting2 := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: setting.PrincipalARN,
					Name:         setting.Name,
					Value:        "disabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}

				err = store.AccountSettingStore().Upsert(ctx, setting2)
				Expect(err).NotTo(HaveOccurred())

				// Verify it was updated
				retrieved, err := store.AccountSettingStore().Get(ctx, setting2.PrincipalARN, setting2.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Value).To(Equal("disabled"))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing account setting", func() {
			It("should return the setting", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/test-get",
					Name:         "containerInstanceLongArnFormat",
					Value:        "enabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}
				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := store.AccountSettingStore().Get(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.PrincipalARN).To(Equal(setting.PrincipalARN))
				Expect(retrieved.Name).To(Equal(setting.Name))
				Expect(retrieved.Value).To(Equal(setting.Value))
			})
		})

		Context("when getting a non-existent account setting", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.AccountSettingStore().Get(ctx, "arn:aws:iam::000000000000:user/non-existent", "serviceLongArnFormat")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing account setting", func() {
			It("should update the setting successfully", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/test-update",
					Name:         "taskLongArnFormat",
					Value:        "disabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}
				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				// Update setting value using Upsert
				setting.Value = "enabled"
				err = store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.AccountSettingStore().Get(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Value).To(Equal("enabled"))
			})
		})

		Context("when updating a non-existent account setting", func() {
			It("should return ErrResourceNotFound", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/non-existent",
					Name:         "taskLongArnFormat",
					Value:        "enabled",
				}

				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				// Verify it was created
				retrieved, err := store.AccountSettingStore().Get(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Value).To(Equal("enabled"))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing account setting", func() {
			It("should delete the setting successfully", func() {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/test-delete",
					Name:         "containerLongArnFormat",
					Value:        "enabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}
				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())

				err = store.AccountSettingStore().Delete(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.AccountSettingStore().Get(ctx, setting.PrincipalARN, setting.Name)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})

		Context("when deleting a non-existent account setting", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.AccountSettingStore().Delete(ctx, "arn:aws:iam::000000000000:user/non-existent", "taskLongArnFormat")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create test settings
			settingNames := []string{"serviceLongArnFormat", "taskLongArnFormat", "containerInstanceLongArnFormat"}
			for _, name := range settingNames {
				setting := &storage.AccountSetting{
					ID:           uuid.New().String(),
					PrincipalARN: "arn:aws:iam::000000000000:user/test-list",
					Name:         name,
					Value:        "enabled",
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}
				err := store.AccountSettingStore().Upsert(ctx, setting)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when listing account settings for a principal", func() {
			It("should return all settings", func() {
				settings, _, err := store.AccountSettingStore().List(ctx, storage.AccountSettingFilters{
					PrincipalARN: "arn:aws:iam::000000000000:user/test-list",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).To(HaveLen(3))
			})
		})

		Context("when listing settings for a non-existent principal", func() {
			It("should return empty list", func() {
				settings, _, err := store.AccountSettingStore().List(ctx, storage.AccountSettingFilters{
					PrincipalARN: "arn:aws:iam::000000000000:user/no-settings",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).To(BeEmpty())
			})
		})
	})
})
