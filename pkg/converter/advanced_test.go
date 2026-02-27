package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestConvertContainer_Advanced(t *testing.T) {
	input := `
[Container]
Image=nginx
Memory=512Mi
User=1000
Group=2000
ReadOnly=true
NoNewPrivileges=true
HealthCmd=curl -f http://localhost
HealthInterval=30s
HealthRetries=3
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qContainer := quadlet.LoadContainer(unit)

	objs, err := ConvertContainer(qContainer, "advanced", nil)
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

	c := deployment.Spec.Template.Spec.Containers[0]

	// Check Resources
	memLimit := c.Resources.Limits[corev1.ResourceMemory]
	if memLimit.String() != "512Mi" {
		t.Errorf("Expected Memory limit 512Mi, got %s", memLimit.String())
	}

	// Check SecurityContext
	sc := c.SecurityContext
	if sc == nil {
		t.Fatal("SecurityContext is nil")
	}
	if sc.RunAsUser == nil {
		t.Fatal("RunAsUser is nil")
	}
	if *sc.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser 1000, got %d", *sc.RunAsUser)
	}
	if sc.RunAsGroup == nil {
		t.Fatal("RunAsGroup is nil")
	}
	if *sc.RunAsGroup != 2000 {
		t.Errorf("Expected RunAsGroup 2000, got %d", *sc.RunAsGroup)
	}
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem true")
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation { // Should be false
		t.Error("Expected AllowPrivilegeEscalation false")
	}

	// Check Probe
	probe := c.LivenessProbe
	if probe == nil {
		t.Fatal("LivenessProbe is nil")
	}
	if probe.Exec == nil || len(probe.Exec.Command) != 3 {
		t.Errorf("Expected Exec command with 3 args, got %v", probe.Exec)
	}
	if probe.PeriodSeconds != 30 {
		t.Errorf("Expected PeriodSeconds 30, got %d", probe.PeriodSeconds)
	}
	if probe.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold 3, got %d", probe.FailureThreshold)
	}
}
