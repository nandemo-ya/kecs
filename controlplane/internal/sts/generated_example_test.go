package generated_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sts "github.com/nandemo-ya/kecs/controlplane/internal/sts/generated"
)

func TestGeneratedTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "STS Generated Types Suite")
}

var _ = Describe("STS Generated Types", func() {
	Describe("AssumeRoleRequest", func() {
		Context("when marshaling to JSON", func() {
			It("should use PascalCase field names", func() {
				req := &sts.AssumeRoleRequest{
					RoleArn:         "arn:aws:iam::123456789012:role/MyRole",
					RoleSessionName: "MySession",
					DurationSeconds: int32Ptr(3600),
					ExternalId:      stringPtr("unique-external-id"),
					Tags: []sts.Tag{
						{
							Key:   "Environment",
							Value: "Test",
						},
					},
				}

				// Marshal to JSON
				data, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				// Verify JSON has camelCase fields
				var jsonMap map[string]interface{}
				err = json.Unmarshal(data, &jsonMap)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonMap["RoleArn"]).To(Equal("arn:aws:iam::123456789012:role/MyRole"))
				Expect(jsonMap["RoleSessionName"]).To(Equal("MySession"))
				Expect(jsonMap["DurationSeconds"]).To(Equal(float64(3600)))
				Expect(jsonMap["ExternalId"]).To(Equal("unique-external-id"))

				// Verify tags
				tags := jsonMap["Tags"].([]interface{})
				Expect(tags).To(HaveLen(1))
				tag := tags[0].(map[string]interface{})
				Expect(tag["Key"]).To(Equal("Environment"))
				Expect(tag["Value"]).To(Equal("Test"))
			})
		})
	})

	Describe("AssumeRoleResponse", func() {
		Context("when unmarshaling from JSON", func() {
			It("should correctly parse PascalCase fields", func() {
				jsonData := `{
			"credentials": {
				"accessKeyId": "AKIAIOSFODNN7EXAMPLE",
				"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				"sessionToken": "AQoDYXdzEGcaEXAMPLE",
				"expiration": "2023-01-01T00:00:00Z"
			},
			"assumedRoleUser": {
				"arn": "arn:aws:sts::123456789012:assumed-role/MyRole/MySession",
				"assumedRoleId": "AROA1234567890EXAMPLE:MySession"
			},
			"packedPolicySize": 6
		}`

				var resp sts.AssumeRoleResponse
				err := json.Unmarshal([]byte(jsonData), &resp)
				Expect(err).NotTo(HaveOccurred())

				// Verify credentials
				Expect(resp.Credentials).NotTo(BeNil())
				Expect(resp.Credentials.AccessKeyId).To(Equal("AKIAIOSFODNN7EXAMPLE"))
				Expect(resp.Credentials.SecretAccessKey).To(Equal("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"))
				Expect(resp.Credentials.SessionToken).To(Equal("AQoDYXdzEGcaEXAMPLE"))

				// Verify assumed role user
				Expect(resp.AssumedRoleUser).NotTo(BeNil())
				Expect(resp.AssumedRoleUser.Arn).To(Equal("arn:aws:sts::123456789012:assumed-role/MyRole/MySession"))
				Expect(resp.AssumedRoleUser.AssumedRoleId).To(Equal("AROA1234567890EXAMPLE:MySession"))

				// Verify packed policy size
				Expect(resp.PackedPolicySize).NotTo(BeNil())
				Expect(*resp.PackedPolicySize).To(Equal(int32(6)))
			})
		})
	})
})

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
