package api

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// Mock implementations for testing

type mockELBv2Integration struct {
	mock.Mock
}

func (m *mockELBv2Integration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*elbv2.LoadBalancer, error) {
	args := m.Called(ctx, name, subnets, securityGroups)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*elbv2.LoadBalancer), args.Error(1)
}

func (m *mockELBv2Integration) DeleteLoadBalancer(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Integration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*elbv2.TargetGroup, error) {
	args := m.Called(ctx, name, port, protocol, vpcId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*elbv2.TargetGroup), args.Error(1)
}

func (m *mockELBv2Integration) DeleteTargetGroup(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Integration) RegisterTargets(ctx context.Context, targetGroupArn string, targets []elbv2.Target) error {
	args := m.Called(ctx, targetGroupArn, targets)
	return args.Error(0)
}

func (m *mockELBv2Integration) DeregisterTargets(ctx context.Context, targetGroupArn string, targets []elbv2.Target) error {
	args := m.Called(ctx, targetGroupArn, targets)
	return args.Error(0)
}

func (m *mockELBv2Integration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*elbv2.Listener, error) {
	args := m.Called(ctx, loadBalancerArn, port, protocol, targetGroupArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*elbv2.Listener), args.Error(1)
}

func (m *mockELBv2Integration) DeleteListener(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Integration) GetLoadBalancer(ctx context.Context, arn string) (*elbv2.LoadBalancer, error) {
	args := m.Called(ctx, arn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*elbv2.LoadBalancer), args.Error(1)
}

func (m *mockELBv2Integration) GetTargetHealth(ctx context.Context, targetGroupArn string) ([]elbv2.TargetHealth, error) {
	args := m.Called(ctx, targetGroupArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]elbv2.TargetHealth), args.Error(1)
}

func (m *mockELBv2Integration) CheckTargetHealthWithK8s(ctx context.Context, targetIP string, targetPort int32, targetGroupArn string) (string, error) {
	args := m.Called(ctx, targetIP, targetPort, targetGroupArn)
	return args.String(0), args.Error(1)
}

type mockELBv2Store struct {
	mock.Mock
}

func (m *mockELBv2Store) CreateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	args := m.Called(ctx, lb)
	return args.Error(0)
}

func (m *mockELBv2Store) GetLoadBalancer(ctx context.Context, arn string) (*storage.ELBv2LoadBalancer, error) {
	args := m.Called(ctx, arn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2LoadBalancer), args.Error(1)
}

func (m *mockELBv2Store) GetLoadBalancerByARN(ctx context.Context, arn string) (*storage.ELBv2LoadBalancer, error) {
	return m.GetLoadBalancer(ctx, arn)
}

func (m *mockELBv2Store) GetLoadBalancerByName(ctx context.Context, name string) (*storage.ELBv2LoadBalancer, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2LoadBalancer), args.Error(1)
}

func (m *mockELBv2Store) ListLoadBalancers(ctx context.Context, region string) ([]*storage.ELBv2LoadBalancer, error) {
	args := m.Called(ctx, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2LoadBalancer), args.Error(1)
}

func (m *mockELBv2Store) UpdateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	args := m.Called(ctx, lb)
	return args.Error(0)
}

func (m *mockELBv2Store) SaveLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	return m.UpdateLoadBalancer(ctx, lb)
}

func (m *mockELBv2Store) DeleteLoadBalancer(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Store) CreateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	args := m.Called(ctx, tg)
	return args.Error(0)
}

func (m *mockELBv2Store) GetTargetGroup(ctx context.Context, arn string) (*storage.ELBv2TargetGroup, error) {
	args := m.Called(ctx, arn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2TargetGroup), args.Error(1)
}

func (m *mockELBv2Store) GetTargetGroupByARN(ctx context.Context, arn string) (*storage.ELBv2TargetGroup, error) {
	return m.GetTargetGroup(ctx, arn)
}

func (m *mockELBv2Store) GetTargetGroupByName(ctx context.Context, name string) (*storage.ELBv2TargetGroup, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2TargetGroup), args.Error(1)
}

func (m *mockELBv2Store) ListTargetGroups(ctx context.Context, region string) ([]*storage.ELBv2TargetGroup, error) {
	args := m.Called(ctx, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2TargetGroup), args.Error(1)
}

func (m *mockELBv2Store) UpdateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	args := m.Called(ctx, tg)
	return args.Error(0)
}

func (m *mockELBv2Store) SaveTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	return m.UpdateTargetGroup(ctx, tg)
}

func (m *mockELBv2Store) DeleteTargetGroup(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Store) CreateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	args := m.Called(ctx, listener)
	return args.Error(0)
}

func (m *mockELBv2Store) GetListener(ctx context.Context, arn string) (*storage.ELBv2Listener, error) {
	args := m.Called(ctx, arn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2Listener), args.Error(1)
}

func (m *mockELBv2Store) GetListenerByARN(ctx context.Context, arn string) (*storage.ELBv2Listener, error) {
	return m.GetListener(ctx, arn)
}

func (m *mockELBv2Store) ListListenersByLoadBalancer(ctx context.Context, loadBalancerArn string) ([]*storage.ELBv2Listener, error) {
	args := m.Called(ctx, loadBalancerArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2Listener), args.Error(1)
}

func (m *mockELBv2Store) ListListeners(ctx context.Context, region string) ([]*storage.ELBv2Listener, error) {
	args := m.Called(ctx, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2Listener), args.Error(1)
}

func (m *mockELBv2Store) UpdateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	args := m.Called(ctx, listener)
	return args.Error(0)
}

func (m *mockELBv2Store) SaveListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	return m.UpdateListener(ctx, listener)
}

func (m *mockELBv2Store) DeleteListener(ctx context.Context, arn string) error {
	args := m.Called(ctx, arn)
	return args.Error(0)
}

func (m *mockELBv2Store) RegisterTargets(ctx context.Context, targetGroupArn string, targets []*storage.ELBv2Target) error {
	args := m.Called(ctx, targetGroupArn, targets)
	return args.Error(0)
}

func (m *mockELBv2Store) DeregisterTargets(ctx context.Context, targetGroupArn string, targetIds []string) error {
	args := m.Called(ctx, targetGroupArn, targetIds)
	return args.Error(0)
}

func (m *mockELBv2Store) GetTargets(ctx context.Context, targetGroupArn string) ([]*storage.ELBv2Target, error) {
	args := m.Called(ctx, targetGroupArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2Target), args.Error(1)
}

func (m *mockELBv2Store) ListTargets(ctx context.Context, targetGroupArn string) ([]*storage.ELBv2Target, error) {
	args := m.Called(ctx, targetGroupArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2Target), args.Error(1)
}

func (m *mockELBv2Store) SaveTargetHealth(ctx context.Context, targetGroupArn string, targetId string, health *storage.ELBv2TargetHealth) error {
	args := m.Called(ctx, targetGroupArn, targetId, health)
	return args.Error(0)
}

func (m *mockELBv2Store) GetTargetHealth(ctx context.Context, targetGroupArn string, targetId string) (*storage.ELBv2TargetHealth, error) {
	args := m.Called(ctx, targetGroupArn, targetId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2TargetHealth), args.Error(1)
}

func (m *mockELBv2Store) ListTargetHealthByTargetGroup(ctx context.Context, targetGroupArn string) ([]*storage.ELBv2TargetHealth, error) {
	args := m.Called(ctx, targetGroupArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2TargetHealth), args.Error(1)
}

func (m *mockELBv2Store) UpdateTargetHealth(ctx context.Context, targetGroupArn string, targetId string, health *storage.ELBv2TargetHealth) error {
	args := m.Called(ctx, targetGroupArn, targetId, health)
	return args.Error(0)
}

func (m *mockELBv2Store) DeleteTargetHealth(ctx context.Context, targetGroupArn string, targetId string) error {
	args := m.Called(ctx, targetGroupArn, targetId)
	return args.Error(0)
}

// Rule operations
func (m *mockELBv2Store) CreateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockELBv2Store) GetRule(ctx context.Context, ruleArn string) (*storage.ELBv2Rule, error) {
	args := m.Called(ctx, ruleArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ELBv2Rule), args.Error(1)
}

func (m *mockELBv2Store) ListRules(ctx context.Context, listenerArn string) ([]*storage.ELBv2Rule, error) {
	args := m.Called(ctx, listenerArn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ELBv2Rule), args.Error(1)
}

func (m *mockELBv2Store) UpdateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockELBv2Store) DeleteRule(ctx context.Context, ruleArn string) error {
	args := m.Called(ctx, ruleArn)
	return args.Error(0)
}

type mockStorage struct {
	mock.Mock
	elbv2Store storage.ELBv2Store
}

func (m *mockStorage) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockStorage) ClusterStore() storage.ClusterStore {
	return nil
}

func (m *mockStorage) ServiceStore() storage.ServiceStore {
	return nil
}

func (m *mockStorage) TaskStore() storage.TaskStore {
	return nil
}

func (m *mockStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return nil
}

func (m *mockStorage) AccountSettingStore() storage.AccountSettingStore {
	return nil
}

func (m *mockStorage) TaskSetStore() storage.TaskSetStore {
	return nil
}

func (m *mockStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return nil
}

func (m *mockStorage) AttributeStore() storage.AttributeStore {
	return nil
}

func (m *mockStorage) ELBv2Store() storage.ELBv2Store {
	return m.elbv2Store
}

func (m *mockStorage) TaskLogStore() storage.TaskLogStore {
	return nil
}

func (m *mockStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return nil, nil
}

func (m *mockStorage) Close() error {
	return nil
}

var _ = Describe("ELBv2APIImpl", func() {
	var (
		ctx             context.Context
		mockStore       *mockELBv2Store
		mockSt          *mockStorage
		mockIntegration *mockELBv2Integration
		api             *ELBv2APIImpl
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockStore = new(mockELBv2Store)
		mockSt = &mockStorage{elbv2Store: mockStore}
		mockIntegration = new(mockELBv2Integration)

		api = &ELBv2APIImpl{
			storage:          mockSt,
			elbv2Integration: mockIntegration,
			region:           "us-east-1",
			accountID:        "123456789012",
		}
	})

	Describe("convertHealthStateToEnum", func() {
		It("should convert healthy state correctly", func() {
			result := api.convertHealthStateToEnum(TargetHealthStateHealthy)
			Expect(result).To(Equal(generated_elbv2.TargetHealthStateEnumHEALTHY))
		})

		It("should convert unhealthy state correctly", func() {
			result := api.convertHealthStateToEnum(TargetHealthStateUnhealthy)
			Expect(result).To(Equal(generated_elbv2.TargetHealthStateEnumUNHEALTHY))
		})

		It("should convert initial state correctly", func() {
			result := api.convertHealthStateToEnum(TargetHealthStateInitial)
			Expect(result).To(Equal(generated_elbv2.TargetHealthStateEnumINITIAL))
		})

		It("should handle unknown state as unavailable", func() {
			result := api.convertHealthStateToEnum("unknown")
			Expect(result).To(Equal(generated_elbv2.TargetHealthStateEnumUNAVAILABLE))
		})
	})

	Describe("Phase 3: CreateTargetGroup with Kubernetes integration", func() {
		Context("when creating a target group", func() {
			It("should successfully create with k8s resources", func() {
				input := &generated_elbv2.CreateTargetGroupInput{
					Name:     "test-tg",
					Port:     utils.Ptr(int32(80)),
					Protocol: ptrProtocol("HTTP"),
					VpcId:    utils.Ptr("vpc-12345"),
				}

				// Mock storage check - target group doesn't exist
				mockStore.On("GetTargetGroupByName", ctx, "test-tg").Return(nil, nil).Once()

				// Mock Kubernetes integration call
				mockIntegration.On("CreateTargetGroup", ctx, "test-tg", int32(80), "HTTP", "vpc-12345").
					Return(&elbv2.TargetGroup{
						Arn:      "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456",
						Name:     "test-tg",
						Port:     80,
						Protocol: "HTTP",
						VpcId:    "vpc-12345",
					}, nil).Once()

				// Mock storage save
				mockStore.On("CreateTargetGroup", ctx, mock.MatchedBy(func(tg *storage.ELBv2TargetGroup) bool {
					return tg.Name == "test-tg" && tg.Port == 80
				})).Return(nil).Once()

				output, err := api.CreateTargetGroup(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TargetGroups).To(HaveLen(1))
				Expect(*output.TargetGroups[0].TargetGroupName).To(Equal("test-tg"))

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})

			It("should fail when k8s integration fails", func() {
				input := &generated_elbv2.CreateTargetGroupInput{
					Name:     "test-tg-fail",
					Port:     utils.Ptr(int32(8080)),
					Protocol: ptrProtocol("HTTP"),
					VpcId:    utils.Ptr("vpc-12345"),
				}

				mockStore.On("GetTargetGroupByName", ctx, "test-tg-fail").Return(nil, nil).Once()

				// Mock storage save that will succeed
				mockStore.On("CreateTargetGroup", ctx, mock.MatchedBy(func(tg *storage.ELBv2TargetGroup) bool {
					return tg.Name == "test-tg-fail" && tg.Port == 8080
				})).Return(nil).Once()

				// Mock Kubernetes integration failure
				mockIntegration.On("CreateTargetGroup", ctx, "test-tg-fail", int32(8080), "HTTP", "vpc-12345").
					Return(nil, fmt.Errorf("failed to create k8s resources")).Once()

				output, err := api.CreateTargetGroup(ctx, input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create target group in Kubernetes"))
				Expect(output).To(BeNil())

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Phase 3: CreateListener with Kubernetes integration", func() {
		Context("when creating a listener", func() {
			It("should successfully create with traefik update", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				loadBalancerArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456"
				input := &generated_elbv2.CreateListenerInput{
					LoadBalancerArn: loadBalancerArn,
					Port:            utils.Ptr(int32(80)),
					Protocol:        ptrProtocol("HTTP"),
					DefaultActions: []generated_elbv2.Action{
						{
							Type:           generated_elbv2.ActionTypeEnum("forward"),
							TargetGroupArn: &targetGroupArn,
						},
					},
				}

				// Mock storage check - load balancer exists
				mockStore.On("GetLoadBalancer", ctx, loadBalancerArn).
					Return(&storage.ELBv2LoadBalancer{
						ARN:   loadBalancerArn,
						Name:  "test-lb",
						State: "active",
					}, nil).Once()

				// Mock Kubernetes integration call
				mockIntegration.On("CreateListener", ctx, loadBalancerArn, int32(80), "HTTP", targetGroupArn).
					Return(&elbv2.Listener{
						Arn:             "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/test-lb/123456/789",
						LoadBalancerArn: loadBalancerArn,
						Port:            80,
						Protocol:        "HTTP",
					}, nil).Once()

				// Mock storage save
				mockStore.On("CreateListener", ctx, mock.MatchedBy(func(l *storage.ELBv2Listener) bool {
					return l.Port == 80 && l.Protocol == "HTTP"
				})).Return(nil).Once()

				output, err := api.CreateListener(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.Listeners).To(HaveLen(1))
				Expect(*output.Listeners[0].Port).To(Equal(int32(80)))

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})

			It("should fail when traefik config update fails", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				loadBalancerArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb-fail/123456"
				input := &generated_elbv2.CreateListenerInput{
					LoadBalancerArn: loadBalancerArn,
					Port:            utils.Ptr(int32(443)),
					Protocol:        ptrProtocol("HTTPS"),
					DefaultActions: []generated_elbv2.Action{
						{
							Type:           generated_elbv2.ActionTypeEnum("forward"),
							TargetGroupArn: &targetGroupArn,
						},
					},
				}

				mockStore.On("GetLoadBalancer", ctx, loadBalancerArn).
					Return(&storage.ELBv2LoadBalancer{
						ARN:   loadBalancerArn,
						Name:  "test-lb-fail",
						State: "active",
					}, nil).Once()

				// Mock storage save that will succeed
				mockStore.On("CreateListener", ctx, mock.MatchedBy(func(l *storage.ELBv2Listener) bool {
					return l.Port == 443 && l.Protocol == "HTTPS"
				})).Return(nil).Once()

				// Mock Kubernetes integration failure
				mockIntegration.On("CreateListener", ctx, loadBalancerArn, int32(443), "HTTPS", targetGroupArn).
					Return(nil, fmt.Errorf("failed to update traefik config")).Once()

				output, err := api.CreateListener(ctx, input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create listener in Kubernetes"))
				Expect(output).To(BeNil())

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Phase 3: RegisterTargets with Kubernetes integration", func() {
		Context("when registering targets", func() {
			It("should successfully register targets", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				input := &generated_elbv2.RegisterTargetsInput{
					TargetGroupArn: targetGroupArn,
					Targets: []generated_elbv2.TargetDescription{
						{
							Id:   "10.0.1.10",
							Port: utils.Ptr(int32(80)),
						},
						{
							Id:   "10.0.1.11",
							Port: utils.Ptr(int32(80)),
						},
					},
				}

				// Mock storage register targets
				mockStore.On("RegisterTargets", ctx, targetGroupArn, mock.MatchedBy(func(targets []*storage.ELBv2Target) bool {
					return len(targets) == 2 && targets[0].ID == "10.0.1.10" && targets[1].ID == "10.0.1.11"
				})).Return(nil).Once()

				// Mock Kubernetes integration call
				expectedTargets := []elbv2.Target{
					{Id: "10.0.1.10", Port: 80},
					{Id: "10.0.1.11", Port: 80},
				}
				mockIntegration.On("RegisterTargets", ctx, targetGroupArn, expectedTargets).
					Return(nil).Once()

				output, err := api.RegisterTargets(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Phase 3: DeregisterTargets with Kubernetes integration", func() {
		Context("when deregistering targets", func() {
			It("should successfully deregister targets", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				input := &generated_elbv2.DeregisterTargetsInput{
					TargetGroupArn: targetGroupArn,
					Targets: []generated_elbv2.TargetDescription{
						{
							Id:   "10.0.1.10",
							Port: utils.Ptr(int32(80)),
						},
					},
				}

				// Mock update target health to deregistering
				mockStore.On("UpdateTargetHealth", ctx, targetGroupArn, "10.0.1.10", mock.MatchedBy(func(h *storage.ELBv2TargetHealth) bool {
					return h.State == "deregistering" && h.Reason == "Target.DeregistrationInProgress"
				})).Return(nil).Once()

				// Mock storage deregister targets
				mockStore.On("DeregisterTargets", ctx, targetGroupArn, []string{"10.0.1.10"}).
					Return(nil).Once()

				// Mock Kubernetes integration call
				expectedTargets := []elbv2.Target{
					{Id: "10.0.1.10", Port: 80},
				}
				mockIntegration.On("DeregisterTargets", ctx, targetGroupArn, expectedTargets).
					Return(nil).Once()

				output, err := api.DeregisterTargets(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Phase 3: ModifyListener", func() {
		Context("when modifying a listener", func() {
			It("should successfully update listener properties", func() {
				listenerArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/test-lb/123456/789"
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/new-tg/123456"
				input := &generated_elbv2.ModifyListenerInput{
					ListenerArn: listenerArn,
					Port:        utils.Ptr(int32(8080)),
					Protocol:    ptrProtocol("HTTP"),
					DefaultActions: []generated_elbv2.Action{
						{
							Type:           generated_elbv2.ActionTypeEnum("forward"),
							TargetGroupArn: &targetGroupArn,
						},
					},
				}

				// Mock get existing listener
				existingListener := &storage.ELBv2Listener{
					ARN:             listenerArn,
					LoadBalancerArn: "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456",
					Port:            80,
					Protocol:        "HTTP",
				}
				mockStore.On("GetListener", ctx, listenerArn).Return(existingListener, nil).Once()

				// Mock update listener
				mockStore.On("UpdateListener", ctx, mock.MatchedBy(func(l *storage.ELBv2Listener) bool {
					return l.Port == 8080 && l.Protocol == "HTTP"
				})).Return(nil).Once()

				// Mock Kubernetes integration update
				mockIntegration.On("CreateListener", ctx, existingListener.LoadBalancerArn, int32(8080), "HTTP", targetGroupArn).
					Return(&elbv2.Listener{
						Arn:             listenerArn,
						LoadBalancerArn: existingListener.LoadBalancerArn,
						Port:            8080,
						Protocol:        "HTTP",
					}, nil).Once()

				output, err := api.ModifyListener(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.Listeners).To(HaveLen(1))
				Expect(*output.Listeners[0].Port).To(Equal(int32(8080)))

				mockStore.AssertExpectations(GinkgoT())
				mockIntegration.AssertExpectations(GinkgoT())
			})

			It("should fail when listener not found", func() {
				listenerArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/test-lb/123456/999"
				input := &generated_elbv2.ModifyListenerInput{
					ListenerArn: listenerArn,
					Port:        utils.Ptr(int32(8080)),
				}

				mockStore.On("GetListener", ctx, listenerArn).Return(nil, nil).Once()

				output, err := api.ModifyListener(ctx, input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("listener not found"))
				Expect(output).To(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Phase 3: ModifyTargetGroup", func() {
		Context("when modifying a target group", func() {
			It("should successfully update health check properties", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				input := &generated_elbv2.ModifyTargetGroupInput{
					TargetGroupArn:             targetGroupArn,
					HealthCheckPath:            utils.Ptr("/healthz"),
					HealthCheckIntervalSeconds: utils.Ptr(int32(10)),
					HealthyThresholdCount:      utils.Ptr(int32(3)),
					UnhealthyThresholdCount:    utils.Ptr(int32(2)),
					Matcher: &generated_elbv2.Matcher{
						HttpCode: utils.Ptr("200-299"),
					},
				}

				// Mock get existing target group
				existingTG := &storage.ELBv2TargetGroup{
					ARN:                        targetGroupArn,
					Name:                       "test-tg",
					Port:                       80,
					Protocol:                   "HTTP",
					VpcID:                      "vpc-12345",
					HealthCheckPath:            "/health",
					HealthCheckIntervalSeconds: 30,
					HealthyThresholdCount:      5,
					UnhealthyThresholdCount:    2,
				}
				mockStore.On("GetTargetGroup", ctx, targetGroupArn).Return(existingTG, nil).Once()

				// Mock update target group
				mockStore.On("UpdateTargetGroup", ctx, mock.MatchedBy(func(tg *storage.ELBv2TargetGroup) bool {
					return tg.HealthCheckPath == "/healthz" &&
						tg.HealthCheckIntervalSeconds == 10 &&
						tg.HealthyThresholdCount == 3 &&
						tg.UnhealthyThresholdCount == 2
				})).Return(nil).Once()

				output, err := api.ModifyTargetGroup(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TargetGroups).To(HaveLen(1))
				Expect(*output.TargetGroups[0].HealthCheckPath).To(Equal("/healthz"))
				Expect(*output.TargetGroups[0].HealthCheckIntervalSeconds).To(Equal(int32(10)))

				mockStore.AssertExpectations(GinkgoT())
			})

			It("should fail when target group not found", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/missing-tg/123456"
				input := &generated_elbv2.ModifyTargetGroupInput{
					TargetGroupArn:  targetGroupArn,
					HealthCheckPath: utils.Ptr("/healthz"),
				}

				mockStore.On("GetTargetGroup", ctx, targetGroupArn).Return(nil, nil).Once()

				output, err := api.ModifyTargetGroup(ctx, input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("target group not found"))
				Expect(output).To(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("MatchesHealthCheckResponse", func() {
		It("should match single status code", func() {
			Expect(MatchesHealthCheckResponse(200, "200")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(201, "200")).To(BeFalse())
		})

		It("should match status code range", func() {
			Expect(MatchesHealthCheckResponse(200, "200-299")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(250, "200-299")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(299, "200-299")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(300, "200-299")).To(BeFalse())
			Expect(MatchesHealthCheckResponse(199, "200-299")).To(BeFalse())
		})

		It("should match comma-separated status codes", func() {
			Expect(MatchesHealthCheckResponse(200, "200,202,301")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(202, "200,202,301")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(301, "200,202,301")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(201, "200,202,301")).To(BeFalse())
			Expect(MatchesHealthCheckResponse(404, "200,202,301")).To(BeFalse())
		})

		It("should handle whitespace in matchers", func() {
			Expect(MatchesHealthCheckResponse(200, " 200 ")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(250, " 200 - 299 ")).To(BeTrue())
			Expect(MatchesHealthCheckResponse(202, " 200 , 202 , 301 ")).To(BeTrue())
		})

		It("should handle invalid matchers", func() {
			Expect(MatchesHealthCheckResponse(200, "abc")).To(BeFalse())
			Expect(MatchesHealthCheckResponse(200, "")).To(BeFalse())
			Expect(MatchesHealthCheckResponse(200, "200-")).To(BeFalse())
			Expect(MatchesHealthCheckResponse(200, "-299")).To(BeFalse())
		})
	})

	Describe("DescribeTargetHealth with proper health check configuration", func() {
		Context("when describing target health", func() {
			It("should use target group health check configuration", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				targetGroup := &storage.ELBv2TargetGroup{
					ARN:                        targetGroupArn,
					Name:                       "test-tg",
					Protocol:                   "HTTP",
					Port:                       80,
					HealthCheckEnabled:         true,
					HealthCheckProtocol:        "HTTP",
					HealthCheckPath:            "/healthz",
					HealthCheckPort:            "8080",
					HealthCheckIntervalSeconds: 30,
					HealthCheckTimeoutSeconds:  5,
					HealthyThresholdCount:      3,
					UnhealthyThresholdCount:    2,
					Matcher:                    "200-299",
				}

				targets := []*storage.ELBv2Target{
					{
						TargetGroupArn: targetGroupArn,
						ID:             "10.0.1.10",
						Port:           80,
					},
				}

				// Mock get target group
				mockStore.On("GetTargetGroup", ctx, targetGroupArn).Return(targetGroup, nil).Once()

				// Mock list targets
				mockStore.On("ListTargets", ctx, targetGroupArn).Return(targets, nil).Once()

				// Mock Kubernetes health check
				mockIntegration.On("CheckTargetHealthWithK8s", ctx, "10.0.1.10", int32(8080), targetGroupArn).Return("healthy", nil).Once()

				// Mock update target health
				mockStore.On("UpdateTargetHealth", ctx, targetGroupArn, "10.0.1.10", mock.MatchedBy(func(h *storage.ELBv2TargetHealth) bool {
					// The health state will depend on the actual health check result
					return h.State != "" && h.Reason != "" && h.Description != ""
				})).Return(nil).Once()

				input := &generated_elbv2.DescribeTargetHealthInput{
					TargetGroupArn: targetGroupArn,
				}

				output, err := api.DescribeTargetHealth(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TargetHealthDescriptions).To(HaveLen(1))
				Expect(output.TargetHealthDescriptions[0].Target.Id).To(Equal("10.0.1.10"))

				mockStore.AssertExpectations(GinkgoT())
			})

			It("should handle disabled health checks", func() {
				targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				targetGroup := &storage.ELBv2TargetGroup{
					ARN:                targetGroupArn,
					Name:               "test-tg",
					Protocol:           "HTTP",
					Port:               80,
					HealthCheckEnabled: false, // Health check disabled
				}

				targets := []*storage.ELBv2Target{
					{
						TargetGroupArn: targetGroupArn,
						ID:             "10.0.1.10",
						Port:           80,
					},
				}

				// Mock get target group
				mockStore.On("GetTargetGroup", ctx, targetGroupArn).Return(targetGroup, nil).Once()

				// Mock list targets
				mockStore.On("ListTargets", ctx, targetGroupArn).Return(targets, nil).Once()

				// No need to mock CheckTargetHealthWithK8s since health check is disabled

				// Mock update target health - should be healthy when health check is disabled
				mockStore.On("UpdateTargetHealth", ctx, targetGroupArn, "10.0.1.10", mock.MatchedBy(func(h *storage.ELBv2TargetHealth) bool {
					return h.State == "healthy"
				})).Return(nil).Once()

				input := &generated_elbv2.DescribeTargetHealthInput{
					TargetGroupArn: targetGroupArn,
				}

				output, err := api.DescribeTargetHealth(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TargetHealthDescriptions).To(HaveLen(1))

				mockStore.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Tag Management Operations", func() {
		Context("AddTags", func() {
			It("should add tags to load balancer", func() {
				lbArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456"
				existingLB := &storage.ELBv2LoadBalancer{
					ARN:  lbArn,
					Name: "test-lb",
					Tags: map[string]string{
						"Environment": "test",
					},
				}

				input := &generated_elbv2.AddTagsInput{
					ResourceArns: []string{lbArn},
					Tags: []generated_elbv2.Tag{
						{Key: "Application", Value: utils.Ptr("web")},
						{Key: "Team", Value: utils.Ptr("platform")},
					},
				}

				// Mock get load balancer
				mockStore.On("GetLoadBalancer", ctx, lbArn).Return(existingLB, nil).Once()

				// Mock update load balancer
				mockStore.On("UpdateLoadBalancer", ctx, mock.MatchedBy(func(lb *storage.ELBv2LoadBalancer) bool {
					return lb.Tags["Environment"] == "test" &&
						lb.Tags["Application"] == "web" &&
						lb.Tags["Team"] == "platform"
				})).Return(nil).Once()

				output, err := api.AddTags(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})

			It("should add tags to target group", func() {
				tgArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"
				existingTG := &storage.ELBv2TargetGroup{
					ARN:  tgArn,
					Name: "test-tg",
					Tags: nil, // No existing tags
				}

				input := &generated_elbv2.AddTagsInput{
					ResourceArns: []string{tgArn},
					Tags: []generated_elbv2.Tag{
						{Key: "Application", Value: utils.Ptr("api")},
					},
				}

				// Mock get target group
				mockStore.On("GetTargetGroup", ctx, tgArn).Return(existingTG, nil).Once()

				// Mock update target group
				mockStore.On("UpdateTargetGroup", ctx, mock.MatchedBy(func(tg *storage.ELBv2TargetGroup) bool {
					return tg.Tags["Application"] == "api"
				})).Return(nil).Once()

				output, err := api.AddTags(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})

			It("should fail when resource not found", func() {
				lbArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/missing-lb/123456"
				input := &generated_elbv2.AddTagsInput{
					ResourceArns: []string{lbArn},
					Tags: []generated_elbv2.Tag{
						{Key: "Application", Value: utils.Ptr("web")},
					},
				}

				// Mock get load balancer - not found
				mockStore.On("GetLoadBalancer", ctx, lbArn).Return(nil, nil).Once()

				output, err := api.AddTags(ctx, input)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("load balancer not found"))
				Expect(output).To(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})
		})

		Context("RemoveTags", func() {
			It("should remove tags from load balancer", func() {
				lbArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456"
				existingLB := &storage.ELBv2LoadBalancer{
					ARN:  lbArn,
					Name: "test-lb",
					Tags: map[string]string{
						"Environment": "test",
						"Application": "web",
						"Team":        "platform",
					},
				}

				input := &generated_elbv2.RemoveTagsInput{
					ResourceArns: []string{lbArn},
					TagKeys:      []string{"Application", "Team"},
				}

				// Mock get load balancer
				mockStore.On("GetLoadBalancer", ctx, lbArn).Return(existingLB, nil).Once()

				// Mock update load balancer
				mockStore.On("UpdateLoadBalancer", ctx, mock.MatchedBy(func(lb *storage.ELBv2LoadBalancer) bool {
					_, hasApp := lb.Tags["Application"]
					_, hasTeam := lb.Tags["Team"]
					return lb.Tags["Environment"] == "test" && !hasApp && !hasTeam
				})).Return(nil).Once()

				output, err := api.RemoveTags(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())

				mockStore.AssertExpectations(GinkgoT())
			})
		})

		Context("DescribeTags", func() {
			It("should describe tags for multiple resources", func() {
				lbArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456"
				tgArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/123456"

				lb := &storage.ELBv2LoadBalancer{
					ARN:  lbArn,
					Name: "test-lb",
					Tags: map[string]string{
						"Environment": "test",
						"Application": "web",
					},
				}

				tg := &storage.ELBv2TargetGroup{
					ARN:  tgArn,
					Name: "test-tg",
					Tags: map[string]string{
						"Environment": "test",
						"Service":     "api",
					},
				}

				input := &generated_elbv2.DescribeTagsInput{
					ResourceArns: []string{lbArn, tgArn},
				}

				// Mock get load balancer
				mockStore.On("GetLoadBalancer", ctx, lbArn).Return(lb, nil).Once()

				// Mock get target group
				mockStore.On("GetTargetGroup", ctx, tgArn).Return(tg, nil).Once()

				output, err := api.DescribeTags(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TagDescriptions).To(HaveLen(2))

				// Check load balancer tags
				var lbTags []generated_elbv2.Tag
				for _, td := range output.TagDescriptions {
					if *td.ResourceArn == lbArn {
						lbTags = td.Tags
						break
					}
				}
				Expect(lbTags).To(HaveLen(2))

				// Check target group tags
				var tgTags []generated_elbv2.Tag
				for _, td := range output.TagDescriptions {
					if *td.ResourceArn == tgArn {
						tgTags = td.Tags
						break
					}
				}
				Expect(tgTags).To(HaveLen(2))

				mockStore.AssertExpectations(GinkgoT())
			})

			It("should skip non-existent resources", func() {
				lbArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/123456"
				missingArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/missing-lb/123456"

				lb := &storage.ELBv2LoadBalancer{
					ARN:  lbArn,
					Name: "test-lb",
					Tags: map[string]string{
						"Environment": "test",
					},
				}

				input := &generated_elbv2.DescribeTagsInput{
					ResourceArns: []string{lbArn, missingArn},
				}

				// Mock get load balancer - found
				mockStore.On("GetLoadBalancer", ctx, lbArn).Return(lb, nil).Once()

				// Mock get load balancer - not found
				mockStore.On("GetLoadBalancer", ctx, missingArn).Return(nil, fmt.Errorf("not found")).Once()

				output, err := api.DescribeTags(ctx, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.TagDescriptions).To(HaveLen(1))
				Expect(*output.TagDescriptions[0].ResourceArn).To(Equal(lbArn))

				mockStore.AssertExpectations(GinkgoT())
			})
		})
	})
})

// Helper functions
func ptrProtocol(s string) *generated_elbv2.ProtocolEnum {
	p := generated_elbv2.ProtocolEnum(s)
	return &p
}
