package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestConvertPod_WithContainers(t *testing.T) {
	podInput := `
[Pod]
PublishPort=8080:80
Volume=shared-data:/data
`
	containerInput1 := `
[Container]
Image=nginx
Pod=my-pod.pod
`
	containerInput2 := `
[Container]
Image=sidecar
Pod=my-pod.pod
`

	podReader := strings.NewReader(podInput)
	podUnitParsed, _ := parser.Parse(podReader)
	podUnit := quadlet.LoadPod(podUnitParsed)

	c1Reader := strings.NewReader(containerInput1)
	c1Parsed, _ := parser.Parse(c1Reader)
	c1 := quadlet.LoadContainer(c1Parsed)

	c2Reader := strings.NewReader(containerInput2)
	c2Parsed, _ := parser.Parse(c2Reader)
	c2 := quadlet.LoadContainer(c2Parsed)

	containers := []*quadlet.ContainerUnit{c1, c2}
	// Names usually derived from filename.
	// c1 name: "app", c2 name: "sidecar"
	containerNames := []string{"app", "sidecar"}

	objs, err := ConvertPod(podUnit, containers, containerNames, "my-pod", nil)
	if err != nil {
		t.Fatalf("ConvertPod failed: %v", err)
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

	if len(deployment.Spec.Template.Spec.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(deployment.Spec.Template.Spec.Containers))
	}

	// Check containers
	// app
	foundApp := false
	foundSidecar := false
	for _, c := range deployment.Spec.Template.Spec.Containers {
		if c.Image == "nginx" { foundApp = true }
		if c.Image == "sidecar" { foundSidecar = true }
	}
	if !foundApp || !foundSidecar {
		t.Errorf("Missing expected containers")
	}

	// Check Volumes (from Pod)
	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Errorf("Expected 1 volume from Pod, got %d", len(deployment.Spec.Template.Spec.Volumes))
	} else {
		// Only check name if volume exists
		if deployment.Spec.Template.Spec.Volumes[0].Name != "pod-vol-0" {
			t.Errorf("Expected volume name 'pod-vol-0', got %s", deployment.Spec.Template.Spec.Volumes[0].Name)
		}
	}

	// Check Service (from Pod)
	if service == nil {
		t.Fatal("Service not found")
	}
	if service.Spec.Ports[0].Port != 8080 {
		t.Errorf("Expected Service Port 8080, got %d", service.Spec.Ports[0].Port)
	}
}
