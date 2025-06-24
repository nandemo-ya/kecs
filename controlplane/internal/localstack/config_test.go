package localstack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

var _ = Describe("LocalStack Configuration", func() {
	var config *localstack.Config

	BeforeEach(func() {
		config = localstack.DefaultConfig()
	})

	Describe("DefaultConfig", func() {
		It("should return valid default configuration", func() {
			Expect(config).NotTo(BeNil())
			Expect(config.Enabled).To(BeFalse())
			Expect(config.Services).To(ConsistOf("iam", "logs", "ssm", "secretsmanager", "elbv2", "s3"))
			Expect(config.Persistence).To(BeTrue())
			Expect(config.Image).To(Equal("localstack/localstack"))
			Expect(config.Version).To(Equal("latest"))
			Expect(config.Namespace).To(Equal("aws-services"))
			Expect(config.Port).To(Equal(4566))
			Expect(config.EdgePort).To(Equal(4566))
		})

		It("should have valid resource limits", func() {
			Expect(config.Resources.Memory).To(Equal("2Gi"))
			Expect(config.Resources.CPU).To(Equal("1000m"))
			Expect(config.Resources.StorageSize).To(Equal("10Gi"))
		})
	})

	Describe("Validate", func() {
		Context("when config is valid", func() {
			It("should not return error", func() {
				err := config.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when config is invalid", func() {
			It("should return error for nil config", func() {
				var nilConfig *localstack.Config
				err := nilConfig.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for empty image", func() {
				config.Image = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for empty version", func() {
				config.Version = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for empty namespace", func() {
				config.Namespace = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid namespace", func() {
				config.Namespace = "Invalid-Namespace"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid port", func() {
				config.Port = -1
				err := config.Validate()
				Expect(err).To(HaveOccurred())

				config.Port = 70000
				err = config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid service", func() {
				config.Services = []string{"iam", "invalid-service"}
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid memory limit", func() {
				config.Resources.Memory = "invalid"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid CPU limit", func() {
				config.Resources.CPU = "invalid"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Service Management", func() {
		It("should get services as comma-separated string", func() {
			config.Services = []string{"iam", "s3", "logs"}
			Expect(config.GetServicesString()).To(Equal("iam,s3,logs"))
		})

		It("should set services from comma-separated string", func() {
			err := config.SetServicesFromString("s3,iam,logs")
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Services).To(ConsistOf("s3", "iam", "logs"))
		})

		It("should handle empty service string", func() {
			err := config.SetServicesFromString("")
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Services).To(BeEmpty())
		})

		It("should trim spaces from services", func() {
			err := config.SetServicesFromString(" s3 , iam , logs ")
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Services).To(ConsistOf("s3", "iam", "logs"))
		})

		It("should return error for invalid service in string", func() {
			err := config.SetServicesFromString("s3,invalid-service,iam")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Environment Variables", func() {
		It("should merge additional environment variables", func() {
			config.MergeEnvironment(map[string]string{
				"CUSTOM_VAR": "value",
				"DEBUG":      "1",
			})

			Expect(config.Environment["CUSTOM_VAR"]).To(Equal("value"))
			Expect(config.Environment["DEBUG"]).To(Equal("1"))
		})

		It("should get all environment variables", func() {
			config.Services = []string{"s3", "iam"}
			config.Debug = true
			config.Persistence = false

			envVars := config.GetEnvironmentVars()

			Expect(envVars["SERVICES"]).To(Equal("s3,iam"))
			Expect(envVars["DEBUG"]).To(Equal("1"))
			Expect(envVars["PERSISTENCE"]).To(Equal("0"))
			Expect(envVars["DATA_DIR"]).To(Equal("/var/lib/localstack"))
			Expect(envVars["EDGE_PORT"]).To(Equal("4566"))
		})
	})

	Describe("ProxyConfig", func() {
		var proxyConfig *localstack.ProxyConfig

		BeforeEach(func() {
			proxyConfig = localstack.ProxyConfigWithDefaults("http://localstack:4566")
		})

		It("should create proxy config with defaults", func() {
			Expect(proxyConfig.Mode).To(Equal(localstack.ProxyModeEnvironment))
			Expect(proxyConfig.LocalStackEndpoint).To(Equal("http://localstack:4566"))
			Expect(proxyConfig.FallbackEnabled).To(BeTrue())
			Expect(proxyConfig.FallbackOrder).To(ConsistOf(localstack.ProxyModeSidecar, localstack.ProxyModeEnvironment))
		})

		Context("when validating proxy config", func() {
			It("should pass for valid config", func() {
				err := proxyConfig.Validate()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail for empty endpoint", func() {
				proxyConfig.LocalStackEndpoint = ""
				err := proxyConfig.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should fail for invalid mode", func() {
				proxyConfig.Mode = "invalid"
				err := proxyConfig.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("should fail for invalid fallback mode", func() {
				proxyConfig.FallbackOrder = []localstack.ProxyMode{"invalid"}
				err := proxyConfig.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Service Validation", func() {
		It("should validate known services", func() {
			Expect(localstack.IsValidService("s3")).To(BeTrue())
			Expect(localstack.IsValidService("iam")).To(BeTrue())
			Expect(localstack.IsValidService("logs")).To(BeTrue())
			Expect(localstack.IsValidService("ssm")).To(BeTrue())
			Expect(localstack.IsValidService("secretsmanager")).To(BeTrue())
			Expect(localstack.IsValidService("elbv2")).To(BeTrue())
			Expect(localstack.IsValidService("rds")).To(BeTrue())
			Expect(localstack.IsValidService("dynamodb")).To(BeTrue())
		})

		It("should reject unknown services", func() {
			Expect(localstack.IsValidService("unknown")).To(BeFalse())
			Expect(localstack.IsValidService("invalid-service")).To(BeFalse())
		})
	})

	Describe("Service URL", func() {
		It("should return LocalStack endpoint for any service", func() {
			endpoint := "http://localstack:4566"
			Expect(localstack.GetServiceURL(endpoint, "s3")).To(Equal(endpoint))
			Expect(localstack.GetServiceURL(endpoint, "iam")).To(Equal(endpoint))
		})
	})
})
