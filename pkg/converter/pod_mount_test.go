package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestConvertPod_VolumeMountsPropagation(t *testing.T) {
	podInput := `
[Pod]
Volume=/host/data:/pod/data
`
	containerInput := `
[Container]
Image=alpine
`
	pReader := strings.NewReader(podInput)
	pUnit, _ := parser.Parse(pReader)
	qPod := quadlet.LoadPod(pUnit)

	cReader := strings.NewReader(containerInput)
	cUnit, _ := parser.Parse(cReader)
	qContainer := quadlet.LoadContainer(cUnit)

	containers := []*quadlet.ContainerUnit{qContainer}
	names := []string{"app"}

	objs, err := ConvertPod(qPod, containers, names, "test-pod", nil)
	if err != nil {
		t.Fatalf("ConvertPod failed: %v", err)
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

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	found := false
	for _, m := range container.VolumeMounts {
		if m.MountPath == "/pod/data" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected pod volume mount /pod/data in container, got %v", container.VolumeMounts)
	}
}
