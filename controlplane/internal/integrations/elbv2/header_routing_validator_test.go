package elbv2_test

import (
	"strings"
	
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var _ = Describe("HeaderRoutingValidator", func() {
	var validator *elbv2.HeaderRoutingValidator

	BeforeEach(func() {
		validator = elbv2.NewHeaderRoutingValidator()
	})

	Describe("ValidateHeaderConditions", func() {
		Context("with security-sensitive headers", func() {
			It("should warn about Authorization header", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("Authorization"),
							Values:         []string{"Bearer *"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).To(HaveLen(2)) // Security warning + known header error
				
				// Check for authentication warning
				var hasAuthWarning bool
				for _, issue := range issues {
					if issue.Severity == elbv2.IssueSeverityWarning && strings.Contains(issue.Message, "authentication headers") {
						hasAuthWarning = true
						break
					}
				}
				Expect(hasAuthWarning).To(BeTrue())
			})

			It("should warn about API key headers", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-API-Key"),
							Values:         []string{"*"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).To(HaveLen(3)) // Security warning + wildcard warning + known header error
				
				// Check for security warning
				var hasSecurityWarning bool
				for _, issue := range issues {
					if issue.Message == "Routing based on authentication headers may expose sensitive information" {
						hasSecurityWarning = true
						break
					}
				}
				Expect(hasSecurityWarning).To(BeTrue())
			})

			It("should warn about cookie headers", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("Cookie"),
							Values:         []string{"session=*"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).NotTo(BeEmpty())
				
				// Check for cookie warning
				var hasCookieWarning bool
				for _, issue := range issues {
					if issue.Message == "Routing based on cookies or sessions may cause inconsistent behavior" {
						hasCookieWarning = true
						break
					}
				}
				Expect(hasCookieWarning).To(BeTrue())
			})
		})

		Context("with header name validation", func() {
			It("should accept valid header names", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Custom-Header"),
							Values:         []string{"value1"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).To(BeEmpty())
			})

			It("should reject invalid header names", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("Invalid Header!"),
							Values:         []string{"value1"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).NotTo(BeEmpty())
				
				var hasFormatError bool
				for _, issue := range issues {
					if issue.Severity == elbv2.IssueSeverityError && issue.Message == "Invalid header name format" {
						hasFormatError = true
						break
					}
				}
				Expect(hasFormatError).To(BeTrue())
			})

			It("should suggest X- prefix for custom headers", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("Custom-Header"),
							Values:         []string{"value1"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).To(HaveLen(1))
				Expect(issues[0].Severity).To(Equal(elbv2.IssueSeverityInfo))
				Expect(issues[0].Suggestion).To(ContainSubstring("X-Custom-Header"))
			})
		})

		Context("with header value validation", func() {
			It("should reject values with newlines", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Test"),
							Values:         []string{"value\nwith\nnewlines"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).NotTo(BeEmpty())
				
				var hasInvalidChars bool
				for _, issue := range issues {
					if issue.Message == "Header value contains invalid characters (CR/LF)" {
						hasInvalidChars = true
						break
					}
				}
				Expect(hasInvalidChars).To(BeTrue())
			})

			It("should warn about wildcard only values", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Test"),
							Values:         []string{"*"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).NotTo(BeEmpty())
				
				var hasWildcardWarning bool
				for _, issue := range issues {
					if issue.Message == "Using wildcard '*' matches all values" {
						hasWildcardWarning = true
						break
					}
				}
				Expect(hasWildcardWarning).To(BeTrue())
			})

			It("should warn about complex wildcard patterns", func() {
				conditions := []generated_elbv2.RuleCondition{
					{
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr("X-Test"),
							Values:         []string{"*test*value*pattern*"},
						},
					},
				}

				issues, err := validator.ValidateHeaderConditions(conditions)
				Expect(err).NotTo(HaveOccurred())
				Expect(issues).NotTo(BeEmpty())
				
				var hasComplexityWarning bool
				for _, issue := range issues {
					if issue.Message == "Complex wildcard patterns may impact performance" {
						hasComplexityWarning = true
						break
					}
				}
				Expect(hasComplexityWarning).To(BeTrue())
			})
		})
	})

	Describe("GetHeaderSuggestions", func() {
		It("should provide API version suggestions", func() {
			suggestions := validator.GetHeaderSuggestions("api-version")
			Expect(suggestions).NotTo(BeEmpty())
			
			var hasApiVersionHeader bool
			for _, suggestion := range suggestions {
				if suggestion.Header == "X-API-Version" {
					hasApiVersionHeader = true
					Expect(suggestion.Pattern).To(Equal("2.*"))
					break
				}
			}
			Expect(hasApiVersionHeader).To(BeTrue())
		})

		It("should provide feature flag suggestions", func() {
			suggestions := validator.GetHeaderSuggestions("feature-flag")
			Expect(suggestions).NotTo(BeEmpty())
			
			var hasFeatureFlagHeader bool
			for _, suggestion := range suggestions {
				if suggestion.Header == "X-Feature-Flag" {
					hasFeatureFlagHeader = true
					break
				}
			}
			Expect(hasFeatureFlagHeader).To(BeTrue())
		})

		It("should provide tenant routing suggestions", func() {
			suggestions := validator.GetHeaderSuggestions("tenant")
			Expect(suggestions).NotTo(BeEmpty())
			
			var hasTenantHeader bool
			for _, suggestion := range suggestions {
				if suggestion.Header == "X-Tenant-ID" {
					hasTenantHeader = true
					Expect(suggestion.Pattern).To(Equal("enterprise-*"))
					break
				}
			}
			Expect(hasTenantHeader).To(BeTrue())
		})

		It("should provide mobile routing suggestions", func() {
			suggestions := validator.GetHeaderSuggestions("mobile")
			Expect(suggestions).NotTo(BeEmpty())
			
			var hasUserAgentHeader bool
			for _, suggestion := range suggestions {
				if suggestion.Header == "User-Agent" {
					hasUserAgentHeader = true
					Expect(suggestion.Pattern).To(ContainSubstring("Mobile"))
					break
				}
			}
			Expect(hasUserAgentHeader).To(BeTrue())
		})

		It("should return empty for unknown scenarios", func() {
			suggestions := validator.GetHeaderSuggestions("unknown-scenario")
			Expect(suggestions).To(BeEmpty())
		})
	})

	Describe("AnalyzeHeaderRoutingComplexity", func() {
		It("should analyze simple header rules", func() {
			rules := []generated_elbv2.Rule{
				{
					RuleArn: utils.Ptr("rule1"),
					Conditions: []generated_elbv2.RuleCondition{
						{
							HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
								HttpHeaderName: utils.Ptr("X-API-Version"),
								Values:         []string{"1.0"},
							},
						},
					},
				},
			}

			report := validator.AnalyzeHeaderRoutingComplexity(rules)
			Expect(report.TotalRules).To(Equal(1))
			Expect(report.HeaderBasedRules).To(Equal(1))
			Expect(report.UniqueHeaders).To(HaveLen(1))
			Expect(report.GetComplexityLevel()).To(Equal("Simple"))
		})

		It("should increase complexity for wildcard patterns", func() {
			rules := []generated_elbv2.Rule{
				{
					RuleArn: utils.Ptr("rule1"),
					Conditions: []generated_elbv2.RuleCondition{
						{
							HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
								HttpHeaderName: utils.Ptr("X-API-Version"),
								Values:         []string{"1.*", "2.*", "3.*"},
							},
						},
					},
				},
			}

			report := validator.AnalyzeHeaderRoutingComplexity(rules)
			Expect(report.ComplexityScore).To(BeNumerically(">", 5))
		})

		It("should handle multiple unique headers", func() {
			rules := []generated_elbv2.Rule{
				{
					RuleArn: utils.Ptr("rule1"),
					Conditions: []generated_elbv2.RuleCondition{
						{
							HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
								HttpHeaderName: utils.Ptr("X-API-Version"),
								Values:         []string{"1.0"},
							},
						},
					},
				},
				{
					RuleArn: utils.Ptr("rule2"),
					Conditions: []generated_elbv2.RuleCondition{
						{
							HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
								HttpHeaderName: utils.Ptr("X-Feature-Flag"),
								Values:         []string{"beta"},
							},
						},
					},
				},
				{
					RuleArn: utils.Ptr("rule3"),
					Conditions: []generated_elbv2.RuleCondition{
						{
							HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
								HttpHeaderName: utils.Ptr("User-Agent"),
								Values:         []string{"*Mobile*"},
							},
						},
					},
				},
			}

			report := validator.AnalyzeHeaderRoutingComplexity(rules)
			Expect(report.TotalRules).To(Equal(3))
			Expect(report.HeaderBasedRules).To(Equal(3))
			Expect(report.UniqueHeaders).To(HaveLen(3))
			Expect(report.GetComplexityLevel()).To(Equal("Moderate"))
		})
	})
})