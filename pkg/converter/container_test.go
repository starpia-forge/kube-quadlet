package converter

import (
	"kube-quadlet/pkg/parser"
	"kube-quadlet/pkg/quadlet"
	"reflect"
	"sort"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestConvertContainer_Basic(t *testing.T) {
	input := `
[Unit]
Description=My Nginx

[Container]
Image=nginx:latest
Exec=/bin/sh -c "echo hello"
Environment=FOO=bar
Environment=BAZ=qux
PublishPort=8080:80
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qContainer := quadlet.LoadContainer(unit)

	objs, err := ConvertContainer(qContainer, "my-app")
	if err != nil {
		t.Fatalf("ConvertContainer failed: %v", err)
	}

	var deployment *appsv1.Deployment
	var service *corev1.Service

	for _, obj := range objs {
		switch o := obj.(type) {
		case *appsv1.Deployment:
			deployment = o
		case *corev1.Service:
			service = o
		}
	}

	if deployment == nil {
		t.Fatal("Deployment not found")
	}

	// Verify Deployment
	if deployment.Name != "my-app" {
		t.Errorf("Expected Deployment name 'my-app', got '%s'", deployment.Name)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatal("Expected 1 container")
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != "nginx:latest" {
		t.Errorf("Expected Image 'nginx:latest', got '%s'", container.Image)
	}

	// Verify Args
	expectedArgs := []string{"/bin/sh", "-c", "echo hello"}
	if !reflect.DeepEqual(container.Args, expectedArgs) {
		t.Errorf("Expected Args %v, got %v", expectedArgs, container.Args)
	}

	// Verify Env (Sorted check)
	expectedEnv := []corev1.EnvVar{
		{Name: "BAZ", Value: "qux"},
		{Name: "FOO", Value: "bar"},
	}
	// Sort actual env
	actualEnv := container.Env
	sort.Slice(actualEnv, func(i, j int) bool {
		return actualEnv[i].Name < actualEnv[j].Name
	})

	if !reflect.DeepEqual(actualEnv, expectedEnv) {
		t.Errorf("Expected Env %v, got %v", expectedEnv, actualEnv)
	}

	// Verify Service
	if service == nil {
		t.Fatal("Service not found")
	}
	if service.Name != "my-app" {
		t.Errorf("Expected Service name 'my-app', got '%s'", service.Name)
	}
	if len(service.Spec.Ports) != 1 {
		t.Fatal("Expected 1 port")
	}
	if service.Spec.Ports[0].Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", service.Spec.Ports[0].Port)
	}
	if service.Spec.Ports[0].TargetPort.IntVal != 80 {
		t.Errorf("Expected TargetPort 80, got %d", service.Spec.Ports[0].TargetPort.IntVal)
	}
}
