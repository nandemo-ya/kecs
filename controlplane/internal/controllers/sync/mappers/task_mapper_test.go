package mappers

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMapPodPhaseToTaskStatus(t *testing.T) {
	mapper := NewTaskStateMapper("123456789012", "us-east-1")

	tests := []struct {
		name           string
		pod            *corev1.Pod
		wantDesired    string
		wantLast       string
	}{
		{
			name: "Pod Running with all containers ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: true,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: metav1.NewTime(time.Now()),
								},
							},
						},
					},
				},
			},
			wantDesired: "RUNNING",
			wantLast:    "RUNNING",
		},
		{
			name: "Pod Running with containers running but not ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: metav1.NewTime(time.Now()),
								},
							},
						},
					},
				},
			},
			wantDesired: "RUNNING",
			wantLast:    "RUNNING", // Should be RUNNING, not PENDING
		},
		{
			name: "Pod Running with no running containers",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
			},
			wantDesired: "RUNNING",
			wantLast:    "ACTIVATING",
		},
		{
			name: "Pod Pending with ContainerCreating",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "ContainerCreating",
								},
							},
						},
					},
				},
			},
			wantDesired: "RUNNING",
			wantLast:    "PENDING",
		},
		{
			name: "Pod Pending with no container statuses",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase:             corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			wantDesired: "RUNNING",
			wantLast:    "PROVISIONING",
		},
		{
			name: "Pod Succeeded",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			wantDesired: "STOPPED",
			wantLast:    "STOPPED",
		},
		{
			name: "Pod Failed",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			wantDesired: "STOPPED",
			wantLast:    "STOPPED",
		},
		{
			name: "Pod being deleted",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			wantDesired: "STOPPED",
			wantLast:    "DEPROVISIONING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDesired, gotLast := mapper.MapPodPhaseToTaskStatus(tt.pod)
			if gotDesired != tt.wantDesired {
				t.Errorf("MapPodPhaseToTaskStatus() gotDesired = %v, want %v", gotDesired, tt.wantDesired)
			}
			if gotLast != tt.wantLast {
				t.Errorf("MapPodPhaseToTaskStatus() gotLast = %v, want %v", gotLast, tt.wantLast)
			}
		})
	}
}