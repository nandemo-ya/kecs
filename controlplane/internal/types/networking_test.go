package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("Networking Types", func() {
	Describe("GetNetworkMode", func() {
		It("should return awsvpc as default when nil", func() {
			mode := types.GetNetworkMode(nil)
			Expect(mode).To(Equal(types.NetworkModeAWSVPC))
		})

		It("should return awsvpc as default when empty string", func() {
			empty := ""
			mode := types.GetNetworkMode(&empty)
			Expect(mode).To(Equal(types.NetworkModeAWSVPC))
		})

		It("should return the specified mode", func() {
			bridge := "bridge"
			mode := types.GetNetworkMode(&bridge)
			Expect(mode).To(Equal(types.NetworkModeBridge))

			host := "host"
			mode = types.GetNetworkMode(&host)
			Expect(mode).To(Equal(types.NetworkModeHost))

			none := "none"
			mode = types.GetNetworkMode(&none)
			Expect(mode).To(Equal(types.NetworkModeNone))

			awsvpc := "awsvpc"
			mode = types.GetNetworkMode(&awsvpc)
			Expect(mode).To(Equal(types.NetworkModeAWSVPC))
		})
	})

	Describe("IsValidNetworkMode", func() {
		It("should validate correct network modes", func() {
			Expect(types.IsValidNetworkMode("awsvpc")).To(BeTrue())
			Expect(types.IsValidNetworkMode("bridge")).To(BeTrue())
			Expect(types.IsValidNetworkMode("host")).To(BeTrue())
			Expect(types.IsValidNetworkMode("none")).To(BeTrue())
		})

		It("should reject invalid network modes", func() {
			Expect(types.IsValidNetworkMode("")).To(BeFalse())
			Expect(types.IsValidNetworkMode("invalid")).To(BeFalse())
			Expect(types.IsValidNetworkMode("AWSVPC")).To(BeFalse()) // case sensitive
			Expect(types.IsValidNetworkMode("nat")).To(BeFalse())
		})
	})

	Describe("RequiresNetworkConfiguration", func() {
		It("should return true only for awsvpc mode", func() {
			Expect(types.RequiresNetworkConfiguration(types.NetworkModeAWSVPC)).To(BeTrue())
		})

		It("should return false for other modes", func() {
			Expect(types.RequiresNetworkConfiguration(types.NetworkModeBridge)).To(BeFalse())
			Expect(types.RequiresNetworkConfiguration(types.NetworkModeHost)).To(BeFalse())
			Expect(types.RequiresNetworkConfiguration(types.NetworkModeNone)).To(BeFalse())
		})
	})

	Describe("NetworkConfiguration", func() {
		It("should properly structure awsvpc configuration", func() {
			config := types.NetworkConfiguration{
				AwsvpcConfiguration: &types.AwsvpcConfiguration{
					Subnets:        []string{"subnet-1", "subnet-2"},
					SecurityGroups: []string{"sg-1", "sg-2"},
					AssignPublicIp: types.AssignPublicIpEnabled,
				},
			}

			Expect(config.AwsvpcConfiguration).NotTo(BeNil())
			Expect(config.AwsvpcConfiguration.Subnets).To(HaveLen(2))
			Expect(config.AwsvpcConfiguration.SecurityGroups).To(HaveLen(2))
			Expect(config.AwsvpcConfiguration.AssignPublicIp).To(Equal(types.AssignPublicIpEnabled))
		})
	})

	// NetworkInterface and NetworkBinding tests are in task_test.go

	Describe("ServiceRegistry", func() {
		It("should handle service discovery configuration", func() {
			arn := "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-12345"
			port := 8080
			containerName := "web"
			containerPort := 80

			registry := types.ServiceRegistry{
				RegistryArn:   &arn,
				Port:          &port,
				ContainerName: &containerName,
				ContainerPort: &containerPort,
			}

			Expect(*registry.RegistryArn).To(Equal(arn))
			Expect(*registry.Port).To(Equal(8080))
			Expect(*registry.ContainerName).To(Equal("web"))
			Expect(*registry.ContainerPort).To(Equal(80))
		})
	})

	Describe("LoadBalancer", func() {
		It("should handle load balancer configuration", func() {
			targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/12345"
			lbName := "my-alb"
			containerName := "app"
			containerPort := 3000

			lb := types.LoadBalancer{
				TargetGroupArn:   &targetGroupArn,
				LoadBalancerName: &lbName,
				ContainerName:    &containerName,
				ContainerPort:    &containerPort,
			}

			Expect(*lb.TargetGroupArn).To(Equal(targetGroupArn))
			Expect(*lb.LoadBalancerName).To(Equal("my-alb"))
			Expect(*lb.ContainerName).To(Equal("app"))
			Expect(*lb.ContainerPort).To(Equal(3000))
		})
	})

	Describe("NetworkAnnotations", func() {
		It("should structure Kubernetes annotations correctly", func() {
			annotations := types.NetworkAnnotations{
				NetworkMode:    "awsvpc",
				Subnets:        "subnet-1,subnet-2",
				SecurityGroups: "sg-1,sg-2",
				AssignPublicIp: "ENABLED",
				PrivateIp:      "10.0.0.5",
			}

			Expect(annotations.NetworkMode).To(Equal("awsvpc"))
			Expect(annotations.Subnets).To(Equal("subnet-1,subnet-2"))
			Expect(annotations.SecurityGroups).To(Equal("sg-1,sg-2"))
			Expect(annotations.AssignPublicIp).To(Equal("ENABLED"))
			Expect(annotations.PrivateIp).To(Equal("10.0.0.5"))
		})
	})
})
