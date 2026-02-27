package converter

import (
	"kuadlet/pkg/parser"
	"kuadlet/pkg/quadlet"
	"strings"
	"testing"
)

func TestConvertContainer_DuplicatePorts(t *testing.T) {
	// This should fail because both map to container port 80?
	// Wait, standard K8s allows multiple services pointing to same container port.
	// But `kuadlet` generates a single Service with multiple ports.
	// `PublishPort=host:container`.
	// 8080:80 -> Service Port 8080, Target 80.
	// 9090:80 -> Service Port 9090, Target 80.
	// These are DISTINCT Service Ports. They are valid.

	// The issue reported was: "If a .container defines duplicate container ports... the converter directly pushes these into the K8s Service object without deduplication, causing the Kubernetes API to reject the Service creation due to Duplicate value errors."
	// Wait, `PublishPort=8080:80` and `PublishPort=9090:80` are distinct ports on the host side (8080 vs 9090).
	// K8s Service `ports`:
	// - port: 8080, targetPort: 80
	// - port: 9090, targetPort: 80
	// This is VALID in K8s.

	// The report said: "duplicate container ports (e.g., PublishPort=8080:80 and PublishPort=9090:80)... causing Duplicate value errors."
	// Maybe I misunderstood the report or the report phrasing is tricky.
	// Let's re-read: "If a .container defines duplicate container ports (e.g., PublishPort=8080:80 and PublishPort=9090:80), the converter directly pushes these into the K8s Service object without deduplication".
	// Ah, maybe the "Duplicate value" refers to the *Name*?
	// `parsePortSpec` generates a name `port-%d`. They are unique.

	// Let's look at the "Conflicting Ports Edge Case" in the report:
	// "If a .container defines multiple PublishPort entries that point to the same K8s container port (e.g., 8080:80 and 9090:80)..."
	// "However, if there are duplicate ports mapped identically in the PublishPort configuration..."
	// "duplicate ports mapped identically" implies `PublishPort=8080:80` AND `PublishPort=8080:80`.
	// OR `PublishPort=8080:80` and `PublishPort=8080:90`.
	// If the HOST port (Service Port) is the same, K8s rejects it.

	// So my check `seenServicePorts[sPort.Port]` checks for duplicate HOST ports.
	// Let's test THAT.

	duplicateInput := `
[Container]
Image=nginx
PublishPort=8080:80
PublishPort=8080:90
`
	// Here host port 8080 is used twice. This is invalid for a Service.

	reader := strings.NewReader(duplicateInput)
	unit, _ := parser.Parse(reader)
	qContainer := quadlet.LoadContainer(unit)

	_, err := ConvertContainer(qContainer, "app", nil)
	if err == nil {
		t.Fatal("Expected error for duplicate ports, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate port definition") {
		t.Errorf("Expected duplicate port error, got: %v", err)
	}
}

func TestConvertPod_DuplicatePorts(t *testing.T) {
	input := `
[Pod]
PublishPort=8080:80
PublishPort=8080:81
`
	reader := strings.NewReader(input)
	unit, _ := parser.Parse(reader)
	qPod := quadlet.LoadPod(unit)

	_, err := ConvertPod(qPod, nil, nil, "pod", nil)
	if err == nil {
		t.Fatal("Expected error for duplicate ports, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate port definition") {
		t.Errorf("Expected duplicate port error, got: %v", err)
	}
}
