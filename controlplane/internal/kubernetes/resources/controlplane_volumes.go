package resources

import (
	corev1 "k8s.io/api/core/v1"
)

// createVolumes creates the volumes for the control plane deployment
func createVolumes(config *ControlPlaneConfig) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ControlPlaneConfigMap,
					},
				},
			},
		},
	}

	// Use hostPath if specified, otherwise use PVC
	if config.DataHostPath != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: config.DataHostPath,
					Type: func() *corev1.HostPathType {
						t := corev1.HostPathDirectoryOrCreate
						return &t
					}(),
				},
			},
		})
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ControlPlanePVC,
				},
			},
		})
	}

	return volumes
}
