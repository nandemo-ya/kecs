package converters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("NetworkConverter", func() {
	var (
		converter *converters.NetworkConverter
		region    string
		accountID string
	)

	BeforeEach(func() {
		region = "us-east-1"
		accountID = "123456789012"
		converter = converters.NewNetworkConverter(region, accountID)
	})

	Describe("ConvertNetworkConfiguration", func() {
		Context("with awsvpc configuration", func() {
			It("should convert all fields correctly", func() {
				config := &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345", "subnet-67890"},
						SecurityGroups: []string{"sg-12345", "sg-67890"},
						AssignPublicIp: (*generated.AssignPublicIp)(ptr.String("ENABLED")),
					},
				}

				annotations := converter.ConvertNetworkConfiguration(config, types.NetworkModeAWSVPC)

				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "awsvpc"))
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/subnets", "subnet-12345,subnet-67890"))
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/security-groups", "sg-12345,sg-67890"))
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/assign-public-ip", "ENABLED"))
			})

			It("should handle minimal configuration", func() {
				config := &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets: []string{"subnet-12345"},
					},
				}

				annotations := converter.ConvertNetworkConfiguration(config, types.NetworkModeAWSVPC)

				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "awsvpc"))
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/subnets", "subnet-12345"))
				Expect(annotations).NotTo(HaveKey("ecs.amazonaws.com/security-groups"))
				Expect(annotations).NotTo(HaveKey("ecs.amazonaws.com/assign-public-ip"))
			})

			It("should handle nil configuration", func() {
				annotations := converter.ConvertNetworkConfiguration(nil, types.NetworkModeBridge)

				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "bridge"))
				Expect(annotations).To(HaveLen(1))
			})
		})

		Context("with different network modes", func() {
			It("should set bridge mode", func() {
				annotations := converter.ConvertNetworkConfiguration(nil, types.NetworkModeBridge)
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "bridge"))
			})

			It("should set host mode", func() {
				annotations := converter.ConvertNetworkConfiguration(nil, types.NetworkModeHost)
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "host"))
			})

			It("should set none mode", func() {
				annotations := converter.ConvertNetworkConfiguration(nil, types.NetworkModeNone)
				Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/network-mode", "none"))
			})
		})
	})

	Describe("SerializeNetworkConfig", func() {
		It("should serialize network configuration to JSON", func() {
			config := &generated.NetworkConfiguration{
				AwsvpcConfiguration: &generated.AwsVpcConfiguration{
					Subnets:        []string{"subnet-12345"},
					SecurityGroups: []string{"sg-12345"},
					AssignPublicIp: (*generated.AssignPublicIp)(ptr.String("DISABLED")),
				},
			}

			json, err := converter.SerializeNetworkConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(json).To(ContainSubstring(`"subnets":["subnet-12345"]`))
			Expect(json).To(ContainSubstring(`"securityGroups":["sg-12345"]`))
			Expect(json).To(ContainSubstring(`"assignPublicIp":"DISABLED"`))
		})

		It("should handle nil configuration", func() {
			json, err := converter.SerializeNetworkConfig(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(json).To(BeEmpty())
		})
	})

	Describe("DeserializeNetworkConfig", func() {
		It("should deserialize JSON to network configuration", func() {
			json := `{"awsvpcConfiguration":{"subnets":["subnet-12345"],"securityGroups":["sg-12345"],"assignPublicIp":"ENABLED"}}`

			config, err := converter.DeserializeNetworkConfig(json)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
			Expect(config.AwsvpcConfiguration).NotTo(BeNil())
			Expect(config.AwsvpcConfiguration.Subnets).To(Equal([]string{"subnet-12345"}))
			Expect(config.AwsvpcConfiguration.SecurityGroups).To(Equal([]string{"sg-12345"}))
			Expect(*config.AwsvpcConfiguration.AssignPublicIp).To(Equal(generated.AssignPublicIp("ENABLED")))
		})

		It("should handle empty string", func() {
			config, err := converter.DeserializeNetworkConfig("")
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(BeNil())
		})

		It("should return error for invalid JSON", func() {
			config, err := converter.DeserializeNetworkConfig("invalid json")
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
	})

	Describe("ParseNetworkAnnotations", func() {
		It("should parse all network annotations", func() {
			annotations := map[string]string{
				"ecs.amazonaws.com/network-mode":     "awsvpc",
				"ecs.amazonaws.com/subnets":          "subnet-12345,subnet-67890",
				"ecs.amazonaws.com/security-groups":  "sg-12345,sg-67890",
				"ecs.amazonaws.com/assign-public-ip": "ENABLED",
				"ecs.amazonaws.com/private-ip":       "10.0.0.1",
			}

			parsed := converter.ParseNetworkAnnotations(annotations)
			Expect(parsed).NotTo(BeNil())
			Expect(parsed.NetworkMode).To(Equal("awsvpc"))
			Expect(parsed.Subnets).To(Equal("subnet-12345,subnet-67890"))
			Expect(parsed.SecurityGroups).To(Equal("sg-12345,sg-67890"))
			Expect(parsed.AssignPublicIp).To(Equal("ENABLED"))
			Expect(parsed.PrivateIp).To(Equal("10.0.0.1"))
		})

		It("should handle nil annotations", func() {
			parsed := converter.ParseNetworkAnnotations(nil)
			Expect(parsed).To(BeNil())
		})

		It("should handle partial annotations", func() {
			annotations := map[string]string{
				"ecs.amazonaws.com/network-mode": "bridge",
				"other-annotation":               "value",
			}

			parsed := converter.ParseNetworkAnnotations(annotations)
			Expect(parsed).NotTo(BeNil())
			Expect(parsed.NetworkMode).To(Equal("bridge"))
			Expect(parsed.Subnets).To(BeEmpty())
			Expect(parsed.SecurityGroups).To(BeEmpty())
		})
	})

	Describe("ConvertLoadBalancer", func() {
		It("should convert load balancer configuration", func() {
			lb := &generated.LoadBalancer{
				TargetGroupArn:   ptr.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/1234567890123456"),
				LoadBalancerName: ptr.String("my-lb"),
				ContainerName:    ptr.String("web"),
				ContainerPort:    ptr.Int32(80),
			}

			annotations, port := converter.ConvertLoadBalancer(lb)

			Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/target-group-arn", *lb.TargetGroupArn))
			Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/load-balancer-name", "my-lb"))
			Expect(port).NotTo(BeNil())
			Expect(port.Port).To(Equal(int32(80)))
			Expect(port.TargetPort.IntVal).To(Equal(int32(80)))
			Expect(port.Name).To(Equal("lb-web"))
		})

		It("should handle nil load balancer", func() {
			annotations, port := converter.ConvertLoadBalancer(nil)
			Expect(annotations).To(BeNil())
			Expect(port).To(BeNil())
		})
	})

	Describe("ConvertServiceRegistry", func() {
		It("should convert service registry configuration", func() {
			registry := &generated.ServiceRegistry{
				RegistryArn:   ptr.String("arn:aws:servicediscovery:us-east-1:123456789012:service/srv-12345678"),
				Port:          ptr.Int32(8080),
				ContainerName: ptr.String("app"),
				ContainerPort: ptr.Int32(8080),
			}

			annotations, port := converter.ConvertServiceRegistry(registry, "my-service")

			Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/service-registry-arn", *registry.RegistryArn))
			Expect(port).NotTo(BeNil())
			Expect(port.Port).To(Equal(int32(8080)))
			Expect(port.TargetPort.IntVal).To(Equal(int32(8080)))
			Expect(port.Name).To(Equal("registry-port"))
		})

		It("should handle registry with only container port", func() {
			registry := &generated.ServiceRegistry{
				RegistryArn:   ptr.String("arn:aws:servicediscovery:us-east-1:123456789012:service/srv-12345678"),
				ContainerPort: ptr.Int32(9090),
			}

			annotations, port := converter.ConvertServiceRegistry(registry, "my-service")

			Expect(annotations).To(HaveKeyWithValue("ecs.amazonaws.com/service-registry-arn", *registry.RegistryArn))
			Expect(port).NotTo(BeNil())
			Expect(port.Port).To(Equal(int32(9090)))
			Expect(port.TargetPort.IntVal).To(Equal(int32(9090)))
		})

		It("should handle nil service registry", func() {
			annotations, port := converter.ConvertServiceRegistry(nil, "my-service")
			Expect(annotations).To(BeNil())
			Expect(port).To(BeNil())
		})
	})
})
