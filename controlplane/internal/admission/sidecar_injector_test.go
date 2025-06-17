package admission

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Mock LocalStackManager for testing
type mockLocalStackManager struct {
	enabled         bool
	endpoint        string
	enabledServices []string
}

func (m *mockLocalStackManager) IsEnabled() bool {
	return m.enabled
}

func (m *mockLocalStackManager) GetEndpoint() string {
	return m.endpoint
}

func (m *mockLocalStackManager) GetEnabledServices() []string {
	return m.enabledServices
}

var _ = Describe("SidecarInjector", func() {
	var (
		injector *SidecarInjector
		mockMgr  *mockLocalStackManager
	)

	BeforeEach(func() {
		mockMgr = &mockLocalStackManager{
			enabled:         true,
			endpoint:        "http://localstack:4566",
			enabledServices: []string{"s3", "dynamodb", "sqs"},
		}
		injector = NewSidecarInjector("kecs/aws-proxy:latest", mockMgr)
	})

	Describe("shouldInjectSidecar", func() {
		It("should inject when annotation is set to true", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						InjectSidecarAnnotation: "true",
					},
				},
			}
			Expect(injector.shouldInjectSidecar(pod)).To(BeTrue())
		})

		It("should not inject when annotation is set to false", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						InjectSidecarAnnotation: "false",
					},
				},
			}
			Expect(injector.shouldInjectSidecar(pod)).To(BeFalse())
		})

		It("should inject when container has AWS environment variables", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{Name: "AWS_REGION", Value: "us-east-1"},
								{Name: "AWS_ACCESS_KEY_ID", Value: "test"},
							},
						},
					},
				},
			}
			Expect(injector.shouldInjectSidecar(pod)).To(BeTrue())
		})

		It("should not inject when no AWS variables or annotations", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{Name: "DATABASE_URL", Value: "postgres://localhost"},
							},
						},
					},
				},
			}
			Expect(injector.shouldInjectSidecar(pod)).To(BeFalse())
		})
	})

	Describe("Handle", func() {
		var (
			pod     *corev1.Pod
			podJSON []byte
			req     *admissionv1.AdmissionRequest
		)

		BeforeEach(func() {
			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Annotations: map[string]string{
						InjectSidecarAnnotation: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "myapp:latest",
							Env: []corev1.EnvVar{
								{Name: "AWS_REGION", Value: "us-east-1"},
							},
						},
					},
				},
			}

			var err error
			podJSON, err = json.Marshal(pod)
			Expect(err).NotTo(HaveOccurred())

			req = &admissionv1.AdmissionRequest{
				UID:       "test-uid",
				Name:      "test-pod",
				Namespace: "default",
				Object: runtime.RawExtension{
					Raw: podJSON,
				},
			}
		})

		It("should inject sidecar when LocalStack is enabled", func() {
			resp := injector.Handle(context.Background(), req)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Patch).NotTo(BeNil())

			// Verify patches
			var patches []map[string]interface{}
			err := json.Unmarshal(resp.Patch, &patches)
			Expect(err).NotTo(HaveOccurred())

			// Should have patches for:
			// 1. Adding sidecar container
			// 2. Adding AWS_ENDPOINT_URL env var
			// 3. Adding HTTPS_PROXY env var
			// 4. Adding NO_PROXY env var
			// 5. Adding sidecar-injected annotation
			Expect(len(patches)).To(BeNumerically(">=", 5))

			// Verify sidecar container patch
			var sidecarPatch map[string]interface{}
			for _, patch := range patches {
				if patch["path"] == "/spec/containers/-" {
					sidecarPatch = patch
					break
				}
			}
			Expect(sidecarPatch).NotTo(BeNil())
			Expect(sidecarPatch["op"]).To(Equal("add"))

			container := sidecarPatch["value"].(map[string]interface{})
			Expect(container["name"]).To(Equal(ProxySidecarName))
			Expect(container["image"]).To(Equal("kecs/aws-proxy:latest"))
		})

		It("should skip injection when LocalStack is disabled", func() {
			mockMgr.enabled = false
			resp := injector.Handle(context.Background(), req)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Patch).To(BeNil())
		})

		It("should use custom endpoint from annotation", func() {
			pod.Annotations[LocalStackEndpointAnnotation] = "http://custom-localstack:5000"
			podJSON, _ = json.Marshal(pod)
			req.Object.Raw = podJSON

			resp := injector.Handle(context.Background(), req)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Patch).NotTo(BeNil())

			var patches []map[string]interface{}
			json.Unmarshal(resp.Patch, &patches)

			// Find sidecar container patch
			for _, patch := range patches {
				if patch["path"] == "/spec/containers/-" {
					container := patch["value"].(map[string]interface{})
					envVars := container["env"].([]interface{})
					
					for _, env := range envVars {
						envMap := env.(map[string]interface{})
						if envMap["name"] == "LOCALSTACK_ENDPOINT" {
							Expect(envMap["value"]).To(Equal("http://custom-localstack:5000"))
							return
						}
					}
				}
			}
			Fail("LOCALSTACK_ENDPOINT not found in sidecar environment")
		})

		It("should use custom services from annotation", func() {
			pod.Annotations[ProxyServicesAnnotation] = "s3,rds"
			podJSON, _ = json.Marshal(pod)
			req.Object.Raw = podJSON

			resp := injector.Handle(context.Background(), req)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Patch).NotTo(BeNil())

			var patches []map[string]interface{}
			json.Unmarshal(resp.Patch, &patches)

			// Find sidecar container patch
			for _, patch := range patches {
				if patch["path"] == "/spec/containers/-" {
					container := patch["value"].(map[string]interface{})
					envVars := container["env"].([]interface{})
					
					for _, env := range envVars {
						envMap := env.(map[string]interface{})
						if envMap["name"] == "PROXY_SERVICES" {
							Expect(envMap["value"]).To(Equal("s3,rds"))
							return
						}
					}
				}
			}
			Fail("PROXY_SERVICES not found in sidecar environment")
		})
	})
})

func TestAdmission(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admission Suite")
}