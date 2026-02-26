package quadlet

import (
	"fmt"
	"kube-quadlet/pkg/parser"
	"os"
	"strconv"
	"strings"
)

func LoadContainer(u *parser.Unit) *ContainerUnit {
	return &ContainerUnit{
		Unit:      LoadUnitSection(u),
		Container: LoadContainerSection(u),
		Service:   LoadServiceSection(u),
		Install:   LoadInstallSection(u),
	}
}

func LoadPod(u *parser.Unit) *PodUnit {
	return &PodUnit{
		Unit:    LoadUnitSection(u),
		Pod:     LoadPodSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadVolume(u *parser.Unit) *VolumeUnit {
	return &VolumeUnit{
		Unit:    LoadUnitSection(u),
		Volume:  LoadVolumeSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadUnitSection(u *parser.Unit) UnitSection {
	s := UnitSection{}
	opts := u.Sections["Unit"]
	for _, opt := range opts {
		switch opt.Key {
		case "Description":
			s.Description = opt.Value
		case "Wants":
			s.Wants = append(s.Wants, splitList(opt.Value)...)
		case "Requires":
			s.Requires = append(s.Requires, splitList(opt.Value)...)
		case "After":
			s.After = append(s.After, splitList(opt.Value)...)
		case "Before":
			s.Before = append(s.Before, splitList(opt.Value)...)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Unit]: %s\n", opt.Key)
		}
	}
	return s
}

func LoadServiceSection(u *parser.Unit) ServiceSection {
	s := ServiceSection{}
	opts := u.Sections["Service"]
	for _, opt := range opts {
		switch opt.Key {
		case "Restart":
			s.Restart = opt.Value
		case "TimeoutStartSec":
			s.TimeoutStartSec = opt.Value
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Service]: %s\n", opt.Key)
		}
	}
	return s
}

func LoadInstallSection(u *parser.Unit) InstallSection {
	s := InstallSection{}
	opts := u.Sections["Install"]
	for _, opt := range opts {
		switch opt.Key {
		case "WantedBy":
			s.WantedBy = append(s.WantedBy, splitList(opt.Value)...)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Install]: %s\n", opt.Key)
		}
	}
	return s
}

func LoadContainerSection(u *parser.Unit) ContainerSection {
	c := ContainerSection{
		Environment: make(map[string]string),
		Label:       make(map[string]string),
		Annotation:  make(map[string]string),
	}
	opts := u.Sections["Container"]
	for _, opt := range opts {
		switch opt.Key {
		case "Image":
			c.Image = opt.Value
		case "Exec":
			c.Exec = opt.Value
		case "Entrypoint":
			c.Entrypoint = opt.Value
		case "Environment":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				c.Environment[parts[0]] = parts[1]
			}
		case "EnvironmentFile":
			c.EnvironmentFile = append(c.EnvironmentFile, opt.Value)
		case "PublishPort":
			c.PublishPort = append(c.PublishPort, opt.Value)
		case "Volume":
			c.Volume = append(c.Volume, opt.Value)
		case "User":
			c.User = opt.Value
		case "Group":
			c.Group = opt.Value
		case "WorkingDir":
			c.WorkingDir = opt.Value
		case "Pod":
			c.Pod = opt.Value
		case "Network":
			c.Network = append(c.Network, opt.Value)
		case "NetworkAlias":
			c.NetworkAlias = append(c.NetworkAlias, opt.Value)
		case "HostName":
			c.HostName = opt.Value
		case "HealthCmd":
			c.HealthCmd = opt.Value
		case "HealthInterval":
			c.HealthInterval = opt.Value
		case "HealthRetries":
			if val, err := strconv.Atoi(opt.Value); err == nil {
				c.HealthRetries = val
			}
		case "HealthTimeout":
			c.HealthTimeout = opt.Value
		case "HealthStartPeriod":
			c.HealthStartPeriod = opt.Value
		case "Memory":
			c.Memory = opt.Value
		case "AddCapability":
			c.AddCapability = append(c.AddCapability, splitList(opt.Value)...)
		case "DropCapability":
			c.DropCapability = append(c.DropCapability, splitList(opt.Value)...)
		case "NoNewPrivileges":
			c.NoNewPrivileges = parseBool(opt.Value)
		case "RunInit":
			c.RunInit = parseBool(opt.Value)
		case "ReadOnly":
			c.ReadOnly = parseBool(opt.Value)
		case "Label":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				c.Label[parts[0]] = parts[1]
			}
		case "Annotation":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				c.Annotation[parts[0]] = parts[1]
			}
		case "PodmanArgs":
			c.PodmanArgs = append(c.PodmanArgs, splitArgs(opt.Value)...)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Container]: %s\n", opt.Key)
		}
	}
	return c
}

func LoadPodSection(u *parser.Unit) PodSection {
	p := PodSection{}
	opts := u.Sections["Pod"]
	for _, opt := range opts {
		switch opt.Key {
		case "PodName":
			p.PodName = opt.Value
		case "PublishPort":
			p.PublishPort = append(p.PublishPort, opt.Value)
		case "Volume":
			p.Volume = append(p.Volume, opt.Value)
		case "Network":
			p.Network = append(p.Network, opt.Value)
		case "NetworkAlias":
			p.NetworkAlias = append(p.NetworkAlias, opt.Value)
		case "IP":
			p.IP = opt.Value
		case "GlobalArgs":
			p.GlobalArgs = append(p.GlobalArgs, splitArgs(opt.Value)...)
		case "PodmanArgs":
			p.PodmanArgs = append(p.PodmanArgs, splitArgs(opt.Value)...)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Pod]: %s\n", opt.Key)
		}
	}
	return p
}

func LoadVolumeSection(u *parser.Unit) VolumeSection {
	v := VolumeSection{
		Label: make(map[string]string),
	}
	opts := u.Sections["Volume"]
	for _, opt := range opts {
		switch opt.Key {
		case "VolumeName":
			v.VolumeName = opt.Value
		case "Label":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				v.Label[parts[0]] = parts[1]
			}
		case "User":
			v.User = opt.Value
		case "Group":
			v.Group = opt.Value
		case "Driver":
			v.Driver = opt.Value
		case "Options":
			v.Options = append(v.Options, opt.Value) // Usually just one string like "o=bind,device=/foo"
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Volume]: %s\n", opt.Key)
		}
	}
	return v
}

func splitList(s string) []string {
	// Systemd lists are space separated
	return strings.Fields(s)
}

func splitArgs(s string) []string {
	// Same as list usually, but might handle quotes?
    // Quadlet doesn't fully document parsing of PodmanArgs string.
    // Assuming simple space separation for now.
	return strings.Fields(s)
}

func parseBool(s string) bool {
    // Systemd bools: 1, yes, true, on
    s = strings.ToLower(s)
    return s == "1" || s == "yes" || s == "true" || s == "on"
}
