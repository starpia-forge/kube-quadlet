package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestConvertContainer_VolumeCrossReference(t *testing.T) {
	// 1. Define a volume unit with a custom VolumeName
	volumeInput := `
[Volume]
VolumeName=custom-pvc-name
`
	vReader := strings.NewReader(volumeInput)
	vUnit, _ := parser.Parse(vReader)
	qVolume := quadlet.LoadVolume(vUnit)

	// Registry
	registry := map[string]*quadlet.VolumeUnit{
		"my-data": qVolume,
	}

	// 2. Define a container referencing that volume via filename
	// Filename would be "my-data.volume"
	containerInput := `
[Container]
Image=alpine
Volume=my-data.volume:/data
`
	cReader := strings.NewReader(containerInput)
	cUnit, _ := parser.Parse(cReader)
	qContainer := quadlet.LoadContainer(cUnit)

	// 3. Convert
	objs, err := ConvertContainer(qContainer, "app", registry)
	if err != nil {
		t.Fatalf("ConvertContainer failed: %v", err)
	}

	// 4. Verify PVC ClaimName
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

	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(deployment.Spec.Template.Spec.Volumes))
	}

	vol := deployment.Spec.Template.Spec.Volumes[0]
	if vol.PersistentVolumeClaim == nil {
		t.Fatal("Expected PVC volume source")
	}

	// It should use "custom-pvc-name" from the registry, NOT "my-data"
	if vol.PersistentVolumeClaim.ClaimName != "custom-pvc-name" {
		t.Errorf("Expected ClaimName 'custom-pvc-name', got '%s'", vol.PersistentVolumeClaim.ClaimName)
	}
}

func TestConvertPod_VolumeCrossReference(t *testing.T) {
	// 1. Define a volume unit with a custom VolumeName
	volumeInput := `
[Volume]
VolumeName=pod-pvc-custom
`
	vReader := strings.NewReader(volumeInput)
	vUnit, _ := parser.Parse(vReader)
	qVolume := quadlet.LoadVolume(vUnit)

	// Registry
	registry := map[string]*quadlet.VolumeUnit{
		"pod-vol": qVolume,
	}

	// 2. Define a Pod referencing that volume
	podInput := `
[Pod]
Volume=pod-vol.volume:/data
`
	pReader := strings.NewReader(podInput)
	pUnit, _ := parser.Parse(pReader)
	qPod := quadlet.LoadPod(pUnit)

	// 3. Convert
	objs, err := ConvertPod(qPod, nil, nil, "my-pod", registry)
	if err != nil {
		t.Fatalf("ConvertPod failed: %v", err)
	}

	// 4. Verify PVC ClaimName in Pod Deployment
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

	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(deployment.Spec.Template.Spec.Volumes))
	}

	vol := deployment.Spec.Template.Spec.Volumes[0]
	if vol.PersistentVolumeClaim == nil {
		t.Fatal("Expected PVC volume source")
	}

	// It should use "pod-pvc-custom"
	if vol.PersistentVolumeClaim.ClaimName != "pod-pvc-custom" {
		t.Errorf("Expected ClaimName 'pod-pvc-custom', got '%s'", vol.PersistentVolumeClaim.ClaimName)
	}
}
