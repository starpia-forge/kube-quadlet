package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestConvertContainer_HealthCmd(t *testing.T) {
	input := `
[Container]
Image=alpine
HealthCmd=curl -f http://localhost || exit 1
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qContainer := quadlet.LoadContainer(unit)

	objs, err := ConvertContainer(qContainer, "app", nil)
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

	container := deployment.Spec.Template.Spec.Containers[0]
	probe := container.LivenessProbe

	if probe == nil {
		t.Fatal("LivenessProbe is nil")
	}
	if probe.Exec == nil {
		t.Fatal("Exec is nil")
	}

	// We expect ["sh", "-c", "curl -f http://localhost || exit 1"]
	expected := []string{"sh", "-c", "curl -f http://localhost || exit 1"}
	if !reflect.DeepEqual(probe.Exec.Command, expected) {
		t.Errorf("Expected Exec command %v, got %v", expected, probe.Exec.Command)
	}
}
