package generated_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sm "github.com/nandemo-ya/kecs/controlplane/internal/secretsmanager/generated"
)

func TestGeneratedTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets Manager Generated Types Suite")
}

var _ = Describe("Secrets Manager Generated Types", func() {
	Describe("GetSecretValueRequest", func() {
		Context("when marshaling to JSON", func() {
			It("should use camelCase field names", func() {
		req := &sm.GetSecretValueRequest{
			SecretId:     "my-secret",
			VersionId:    stringPtr("v1"),
			VersionStage: stringPtr("AWSCURRENT"),
		}

				// Marshal to JSON
				data, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				// Verify JSON has camelCase fields
				var jsonMap map[string]interface{}
				err = json.Unmarshal(data, &jsonMap)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonMap["secretId"]).To(Equal("my-secret"))
				Expect(jsonMap["versionId"]).To(Equal("v1"))
				Expect(jsonMap["versionStage"]).To(Equal("AWSCURRENT"))
			})
		})
	})

	Describe("GetSecretValueResponse", func() {
		Context("when unmarshaling from JSON", func() {
			It("should correctly parse camelCase fields", func() {
		jsonData := `{
			"arn": "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			"name": "my-secret",
			"versionId": "12345678-1234-1234-1234-123456789012",
			"secretString": "{\"username\":\"admin\",\"password\":\"secret123\"}",
			"versionStages": ["AWSCURRENT"],
			"createdDate": "2023-01-01T00:00:00Z"
		}`

				var resp sm.GetSecretValueResponse
				err := json.Unmarshal([]byte(jsonData), &resp)
				Expect(err).NotTo(HaveOccurred())

				// Verify fields
				Expect(*resp.ARN).To(Equal("arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf"))
				Expect(*resp.Name).To(Equal("my-secret"))
				Expect(*resp.VersionId).To(Equal("12345678-1234-1234-1234-123456789012"))
				Expect(*resp.SecretString).To(Equal(`{"username":"admin","password":"secret123"}`))
				Expect(resp.VersionStages).To(Equal([]string{"AWSCURRENT"}))
				Expect(resp.CreatedDate).NotTo(BeNil())
			})
		})
	})

	Describe("CreateSecretRequest", func() {
		Context("when marshaling with tags", func() {
			It("should correctly serialize tags with camelCase", func() {
		req := &sm.CreateSecretRequest{
			Name:         "new-secret",
			Description:  stringPtr("My new secret"),
			SecretString: stringPtr(`{"key":"value"}`),
			Tags: []sm.Tag{
				{
					Key:   stringPtr("Environment"),
					Value: stringPtr("Production"),
				},
				{
					Key:   stringPtr("Application"),
					Value: stringPtr("MyApp"),
				},
			},
		}

				// Marshal to JSON
				data, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				// Verify JSON structure
				var jsonMap map[string]interface{}
				err = json.Unmarshal(data, &jsonMap)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonMap["name"]).To(Equal("new-secret"))
				Expect(jsonMap["description"]).To(Equal("My new secret"))
				Expect(jsonMap["secretString"]).To(Equal(`{"key":"value"}`))

				// Verify tags
				tags := jsonMap["tags"].([]interface{})
				Expect(tags).To(HaveLen(2))
				
				tag1 := tags[0].(map[string]interface{})
				Expect(tag1["key"]).To(Equal("Environment"))
				Expect(tag1["value"]).To(Equal("Production"))
				
				tag2 := tags[1].(map[string]interface{})
				Expect(tag2["key"]).To(Equal("Application"))
				Expect(tag2["value"]).To(Equal("MyApp"))
			})
		})
	})
})

// Helper function
func stringPtr(s string) *string {
	return &s
}