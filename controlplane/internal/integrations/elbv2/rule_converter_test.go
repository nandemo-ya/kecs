package elbv2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var _ = Describe("RuleConverter", func() {
	var converter *elbv2.RuleConverter

	BeforeEach(func() {
		converter = elbv2.NewRuleConverter()
	})

	Describe("ConvertRuleToTraefikMatch", func() {
		Context("with path pattern conditions", func() {
			It("should convert exact path", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/api/users"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Path(`/api/users`)"))
			})

			It("should convert path prefix with wildcard", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/api/*"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("PathPrefix(`/api/`)"))
			})

			It("should convert root wildcard path", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/*"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("PathPrefix(`/`)"))
			})

			It("should convert complex wildcard pattern", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/api/*/users"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("PathRegexp(`^/api/.*/users$`)"))
			})

			It("should handle multiple path patterns with OR", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/api/*", "/v1/*", "/health"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(PathPrefix(`/api/`) || PathPrefix(`/v1/`) || Path(`/health`))"))
			})
		})

		Context("with host header conditions", func() {
			It("should convert exact host", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HostHeaderConfig: &generated_elbv2.HostHeaderConditionConfig{
							Values: []string{"api.example.com"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Host(`api.example.com`)"))
			})

			It("should convert wildcard host", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HostHeaderConfig: &generated_elbv2.HostHeaderConditionConfig{
							Values: []string{"*.example.com"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("HostRegexp(`^[^.]+.example.com$`)"))
			})

			It("should handle multiple hosts with OR", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HostHeaderConfig: &generated_elbv2.HostHeaderConditionConfig{
							Values: []string{"api.example.com", "www.example.com"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(Host(`api.example.com`) || Host(`www.example.com`))"))
			})
		})

		Context("with HTTP header conditions", func() {
			It("should convert exact header match", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Custom-Header"),
							Values:         []string{"value1"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Header(`X-Custom-Header`, `value1`)"))
			})

			It("should convert wildcard header value", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Custom-Header"),
							Values:         []string{"prefix-*"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("HeaderRegexp(`X-Custom-Header`, `^prefix-.*$`)"))
			})
		})

		Context("with HTTP method conditions", func() {
			It("should convert single method", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpRequestMethodConfig: &generated_elbv2.HttpRequestMethodConditionConfig{
							Values: []string{"POST"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Method(`POST`)"))
			})

			It("should convert multiple methods with OR", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpRequestMethodConfig: &generated_elbv2.HttpRequestMethodConditionConfig{
							Values: []string{"GET", "POST", "PUT"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(Method(`GET`) || Method(`POST`) || Method(`PUT`))"))
			})
		})

		Context("with query string conditions", func() {
			It("should convert key-value query parameter", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						QueryStringConfig: &generated_elbv2.QueryStringConditionConfig{
							Values: []generated_elbv2.QueryStringKeyValuePair{
								{
									Key:   utils.Ptr("version"),
									Value: utils.Ptr("v2"),
								},
							},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Query(`version=v2`)"))
			})

			It("should convert key-only query parameter", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						QueryStringConfig: &generated_elbv2.QueryStringConditionConfig{
							Values: []generated_elbv2.QueryStringKeyValuePair{
								{
									Key: utils.Ptr("debug"),
								},
							},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Query(`debug`)"))
			})

			It("should combine multiple query parameters with AND", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						QueryStringConfig: &generated_elbv2.QueryStringConditionConfig{
							Values: []generated_elbv2.QueryStringKeyValuePair{
								{
									Key:   utils.Ptr("version"),
									Value: utils.Ptr("v2"),
								},
								{
									Key: utils.Ptr("debug"),
								},
							},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(Query(`version=v2`) && Query(`debug`))"))
			})
		})

		Context("with source IP conditions", func() {
			It("should convert single IP", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						SourceIpConfig: &generated_elbv2.SourceIpConditionConfig{
							Values: []string{"192.168.1.0/24"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("ClientIP(`192.168.1.0/24`)"))
			})

			It("should convert multiple IPs with OR", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						SourceIpConfig: &generated_elbv2.SourceIpConditionConfig{
							Values: []string{"192.168.1.0/24", "10.0.0.0/8"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(ClientIP(`192.168.1.0/24`) || ClientIP(`10.0.0.0/8`))"))
			})
		})

		Context("with multiple condition types", func() {
			It("should combine conditions with AND", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
							Values: []string{"/api/*"},
						},
					},
					{
						HostHeaderConfig: &generated_elbv2.HostHeaderConditionConfig{
							Values: []string{"api.example.com"},
						},
					},
					{
						HttpRequestMethodConfig: &generated_elbv2.HttpRequestMethodConditionConfig{
							Values: []string{"POST"},
						},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("(PathPrefix(`/api/`) && Host(`api.example.com`) && Method(`POST`))"))
			})
		})

		Context("with legacy field-based conditions", func() {
			It("should convert legacy path-pattern field", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						Field:  utils.Ptr("path-pattern"),
						Values: []string{"/api/*"},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("PathPrefix(`/api/`)"))
			})

			It("should convert legacy host-header field", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						Field:  utils.Ptr("host-header"),
						Values: []string{"api.example.com"},
					},
				}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("Host(`api.example.com`)"))
			})
		})

		Context("with empty conditions", func() {
			It("should return default catch-all route", func() {
				conditions := []generated_elbv2.RuleCondition{}

				match, err := converter.ConvertRuleToTraefikMatch(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(match).To(Equal("PathPrefix(`/`)"))
			})
		})
	})

	Describe("ExtractTargetGroupFromActions", func() {
		It("should extract target group from simple forward action", func() {
			actions := []generated_elbv2.Action{
				{
					Type:           generated_elbv2.ActionTypeEnumFORWARD,
					TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067"),
				},
			}

			tgArn, err := converter.ExtractTargetGroupFromActions(actions)
			Expect(err).NotTo(HaveOccurred())
			Expect(tgArn).To(Equal("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067"))
		})

		It("should extract target group from forward config", func() {
			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumFORWARD,
					ForwardConfig: &generated_elbv2.ForwardActionConfig{
						TargetGroups: []generated_elbv2.TargetGroupTuple{
							{
								TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067"),
								Weight:         utils.Ptr(int32(100)),
							},
						},
					},
				},
			}

			tgArn, err := converter.ExtractTargetGroupFromActions(actions)
			Expect(err).NotTo(HaveOccurred())
			Expect(tgArn).To(Equal("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067"))
		})

		It("should return error when no forward action found", func() {
			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumREDIRECT,
				},
			}

			_, err := converter.ExtractTargetGroupFromActions(actions)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no forward action with target group found"))
		})
	})
})