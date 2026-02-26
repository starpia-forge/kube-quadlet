package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
[Unit]
Description=A minimal container
# This is a comment

[Container]
Image=nginx
PublishPort=8080:80
PublishPort=8081:81
Environment=FOO=bar
Environment=BAZ=qux
Exec=/bin/sh -c "echo hello"

[Install]
WantedBy=multi-user.target
`

	expected := &Unit{
		Sections: map[string][]Option{
			"Unit": {
				{Key: "Description", Value: "A minimal container"},
			},
			"Container": {
				{Key: "Image", Value: "nginx"},
				{Key: "PublishPort", Value: "8080:80"},
				{Key: "PublishPort", Value: "8081:81"},
				{Key: "Environment", Value: "FOO=bar"},
				{Key: "Environment", Value: "BAZ=qux"},
				{Key: "Exec", Value: "/bin/sh -c \"echo hello\""},
			},
			"Install": {
				{Key: "WantedBy", Value: "multi-user.target"},
			},
		},
	}

	reader := strings.NewReader(input)
	unit, err := Parse(reader)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !reflect.DeepEqual(unit, expected) {
		t.Errorf("Expected %+v, got %+v", expected, unit)
	}
}

func TestParse_Continuation(t *testing.T) {
    // Systemd supports line continuation with `\`
    input := `
[Service]
ExecStart=/bin/echo \
    one two \
    three
`
    expected := &Unit{
        Sections: map[string][]Option{
            "Service": {
                // Double spaces because input has space before backslash + backslash becomes space
                {Key: "ExecStart", Value: "/bin/echo  one two  three"},
            },
        },
    }

    reader := strings.NewReader(input)
    unit, err := Parse(reader)
    if err != nil {
        t.Fatalf("Parse failed: %v", err)
    }

    // Checking if values are joined by space (standard systemd behavior)
    // Actually systemd replaces the backslash and newline with a space.
    if !reflect.DeepEqual(unit, expected) {
        t.Errorf("Expected %+v, got %+v", expected, unit)
    }
}
