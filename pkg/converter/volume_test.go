package converter

import (
	"kube-quadlet/pkg/parser"
	"kube-quadlet/pkg/quadlet"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
)

func TestConvertVolume(t *testing.T) {
	input := `
[Volume]
VolumeName=my-data
Label=foo=bar
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qVolume := quadlet.LoadVolume(unit)

	objs, err := ConvertVolume(qVolume, "test-vol")
	if err != nil {
		t.Fatalf("ConvertVolume failed: %v", err)
	}

	var pvc *corev1.PersistentVolumeClaim
	for _, obj := range objs {
		if p, ok := obj.(*corev1.PersistentVolumeClaim); ok {
			pvc = p
			break
		}
	}

	if pvc == nil {
		t.Fatal("PVC not found")
	}

	if pvc.Name != "my-data" {
		t.Errorf("Expected PVC name 'my-data', got '%s'", pvc.Name)
	}

	if pvc.Labels["foo"] != "bar" {
		t.Errorf("Expected Label foo=bar, got %v", pvc.Labels)
	}
}

func TestConvertContainer_WithVolume(t *testing.T) {
	input := `
[Container]
Image=busybox
Volume=/host/path:/container/path
Volume=my-pvc:/data
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qContainer := quadlet.LoadContainer(unit)

	objs, err := ConvertContainer(qContainer, "app-with-vol")
	if err != nil {
		t.Fatalf("ConvertContainer failed: %v", err)
	}

	var deployment *appsv1.Deployment
	for _, obj := range objs {
		if d, ok := obj.(*appsv1.Deployment); ok {
			deployment = d
			break
		}
	}

	if deployment == nil {
		t.Fatal("Deployment not found")
	}

	podSpec := deployment.Spec.Template.Spec
	if len(podSpec.Volumes) != 2 {
		t.Fatalf("Expected 2 volumes, got %d", len(podSpec.Volumes))
	}

	// Verify HostPath
	// Since slice order is preserved from input
	vol0 := podSpec.Volumes[0]
	if vol0.HostPath == nil {
		t.Errorf("Expected HostPath for vol0, got %v", vol0.VolumeSource)
	} else if vol0.HostPath.Path != "/host/path" {
		t.Errorf("Expected HostPath /host/path, got %s", vol0.HostPath.Path)
	}

	// Verify PVC
	vol1 := podSpec.Volumes[1]
	if vol1.PersistentVolumeClaim == nil {
		t.Errorf("Expected PVC for vol1, got %v", vol1.VolumeSource)
	} else if vol1.PersistentVolumeClaim.ClaimName != "my-pvc" {
		t.Errorf("Expected ClaimName my-pvc, got %s", vol1.PersistentVolumeClaim.ClaimName)
	}

	// Verify Mounts
	container := podSpec.Containers[0]
	if len(container.VolumeMounts) != 2 {
		t.Fatalf("Expected 2 mounts, got %d", len(container.VolumeMounts))
	}

	mount0 := container.VolumeMounts[0]
	if mount0.MountPath != "/container/path" {
		t.Errorf("Expected MountPath /container/path, got %s", mount0.MountPath)
	}

	mount1 := container.VolumeMounts[1]
	if mount1.MountPath != "/data" {
		t.Errorf("Expected MountPath /data, got %s", mount1.MountPath)
	}
}
