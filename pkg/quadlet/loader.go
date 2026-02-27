package quadlet

import (
	"fmt"
	"kuadlet/pkg/parser"
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

func LoadKube(u *parser.Unit) *KubeUnit {
	return &KubeUnit{
		Unit:    LoadUnitSection(u),
		Kube:    LoadKubeSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadNetwork(u *parser.Unit) *NetworkUnit {
	return &NetworkUnit{
		Unit:    LoadUnitSection(u),
		Network: LoadNetworkSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadImage(u *parser.Unit) *ImageUnit {
	return &ImageUnit{
		Unit:    LoadUnitSection(u),
		Image:   LoadImageSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadBuild(u *parser.Unit) *BuildUnit {
	return &BuildUnit{
		Unit:    LoadUnitSection(u),
		Build:   LoadBuildSection(u),
		Service: LoadServiceSection(u),
		Install: LoadInstallSection(u),
	}
}

func LoadArtifact(u *parser.Unit) *ArtifactUnit {
	return &ArtifactUnit{
		Unit:     LoadUnitSection(u),
		Artifact: LoadArtifactSection(u),
		Service:  LoadServiceSection(u),
		Install:  LoadInstallSection(u),
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

func LoadKubeSection(u *parser.Unit) KubeSection {
	k := KubeSection{}
	opts := u.Sections["Kube"]
	for _, opt := range opts {
		switch opt.Key {
		case "Yaml":
			k.Yaml = opt.Value
		case "AutoUpdate":
			k.AutoUpdate = append(k.AutoUpdate, opt.Value)
		case "ConfigMap":
			k.ConfigMap = append(k.ConfigMap, opt.Value)
		case "ContainersConfModule":
			k.ContainersConfModule = append(k.ContainersConfModule, opt.Value)
		case "ExitCodePropagation":
			k.ExitCodePropagation = opt.Value
		case "GlobalArgs":
			k.GlobalArgs = append(k.GlobalArgs, splitArgs(opt.Value)...)
		case "KubeDownForce":
			k.KubeDownForce = parseBool(opt.Value)
		case "LogDriver":
			k.LogDriver = opt.Value
		case "Network":
			k.Network = append(k.Network, opt.Value)
		case "PodmanArgs":
			k.PodmanArgs = append(k.PodmanArgs, splitArgs(opt.Value)...)
		case "PublishPort":
			k.PublishPort = append(k.PublishPort, opt.Value)
		case "SetWorkingDirectory":
			k.SetWorkingDirectory = opt.Value
		case "UserNS":
			k.UserNS = opt.Value
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Kube]: %s\n", opt.Key)
		}
	}
	return k
}

func LoadNetworkSection(u *parser.Unit) NetworkSection {
	n := NetworkSection{
		Label: make(map[string]string),
	}
	opts := u.Sections["Network"]
	for _, opt := range opts {
		switch opt.Key {
		case "ContainersConfModule":
			n.ContainersConfModule = append(n.ContainersConfModule, opt.Value)
		case "DisableDNS":
			n.DisableDNS = parseBool(opt.Value)
		case "DNS":
			n.DNS = append(n.DNS, opt.Value)
		case "Driver":
			n.Driver = opt.Value
		case "Gateway":
			n.Gateway = append(n.Gateway, opt.Value)
		case "GlobalArgs":
			n.GlobalArgs = append(n.GlobalArgs, splitArgs(opt.Value)...)
		case "InterfaceName":
			n.InterfaceName = opt.Value
		case "Internal":
			n.Internal = parseBool(opt.Value)
		case "IPAMDriver":
			n.IPAMDriver = opt.Value
		case "IPRange":
			n.IPRange = append(n.IPRange, opt.Value)
		case "IPv6":
			n.IPv6 = parseBool(opt.Value)
		case "Label":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				n.Label[parts[0]] = parts[1]
			}
		case "NetworkDeleteOnStop":
			n.NetworkDeleteOnStop = parseBool(opt.Value)
		case "NetworkName":
			n.NetworkName = opt.Value
		case "Options":
			n.Options = append(n.Options, opt.Value)
		case "PodmanArgs":
			n.PodmanArgs = append(n.PodmanArgs, splitArgs(opt.Value)...)
		case "Subnet":
			n.Subnet = append(n.Subnet, opt.Value)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Network]: %s\n", opt.Key)
		}
	}
	return n
}

func LoadImageSection(u *parser.Unit) ImageSection {
	i := ImageSection{}
	opts := u.Sections["Image"]
	for _, opt := range opts {
		switch opt.Key {
		case "AllTags":
			i.AllTags = parseBool(opt.Value)
		case "Arch":
			i.Arch = opt.Value
		case "AuthFile":
			i.AuthFile = opt.Value
		case "CertDir":
			i.CertDir = opt.Value
		case "ContainersConfModule":
			i.ContainersConfModule = append(i.ContainersConfModule, opt.Value)
		case "Creds":
			i.Creds = opt.Value
		case "DecryptionKey":
			i.DecryptionKey = opt.Value
		case "GlobalArgs":
			i.GlobalArgs = append(i.GlobalArgs, splitArgs(opt.Value)...)
		case "Image":
			i.Image = opt.Value
		case "ImageTag":
			i.ImageTag = opt.Value
		case "OS":
			i.OS = opt.Value
		case "PodmanArgs":
			i.PodmanArgs = append(i.PodmanArgs, splitArgs(opt.Value)...)
		case "Policy":
			i.Policy = opt.Value
		case "Retry":
			if val, err := strconv.Atoi(opt.Value); err == nil {
				i.Retry = val
			}
		case "RetryDelay":
			i.RetryDelay = opt.Value
		case "TLSVerify":
			i.TLSVerify = parseBool(opt.Value)
		case "Variant":
			i.Variant = opt.Value
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Image]: %s\n", opt.Key)
		}
	}
	return i
}

func LoadBuildSection(u *parser.Unit) BuildSection {
	b := BuildSection{
		Annotation:  make(map[string]string),
		BuildArg:    make(map[string]string),
		Environment: make(map[string]string),
		Label:       make(map[string]string),
	}
	opts := u.Sections["Build"]
	for _, opt := range opts {
		switch opt.Key {
		case "Annotation":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				b.Annotation[parts[0]] = parts[1]
			}
		case "Arch":
			b.Arch = opt.Value
		case "AuthFile":
			b.AuthFile = opt.Value
		case "BuildArg":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				b.BuildArg[parts[0]] = parts[1]
			}
		case "ContainersConfModule":
			b.ContainersConfModule = append(b.ContainersConfModule, opt.Value)
		case "DNS":
			b.DNS = append(b.DNS, opt.Value)
		case "DNSOption":
			b.DNSOption = append(b.DNSOption, opt.Value)
		case "DNSSearch":
			b.DNSSearch = append(b.DNSSearch, opt.Value)
		case "Environment":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				b.Environment[parts[0]] = parts[1]
			}
		case "File":
			b.File = opt.Value
		case "ForceRM":
			b.ForceRM = parseBool(opt.Value)
		case "GlobalArgs":
			b.GlobalArgs = append(b.GlobalArgs, splitArgs(opt.Value)...)
		case "GroupAdd":
			b.GroupAdd = append(b.GroupAdd, opt.Value)
		case "IgnoreFile":
			b.IgnoreFile = opt.Value
		case "ImageTag":
			b.ImageTag = append(b.ImageTag, opt.Value)
		case "Label":
			parts := strings.SplitN(opt.Value, "=", 2)
			if len(parts) == 2 {
				b.Label[parts[0]] = parts[1]
			}
		case "Network":
			b.Network = append(b.Network, opt.Value)
		case "PodmanArgs":
			b.PodmanArgs = append(b.PodmanArgs, splitArgs(opt.Value)...)
		case "Pull":
			b.Pull = opt.Value
		case "Retry":
			if val, err := strconv.Atoi(opt.Value); err == nil {
				b.Retry = val
			}
		case "RetryDelay":
			b.RetryDelay = opt.Value
		case "Secret":
			b.Secret = append(b.Secret, opt.Value)
		case "SetWorkingDirectory":
			b.SetWorkingDirectory = opt.Value
		case "Target":
			b.Target = opt.Value
		case "TLSVerify":
			b.TLSVerify = parseBool(opt.Value)
		case "Variant":
			b.Variant = opt.Value
		case "Volume":
			b.Volume = append(b.Volume, opt.Value)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Build]: %s\n", opt.Key)
		}
	}
	return b
}

func LoadArtifactSection(u *parser.Unit) ArtifactSection {
	a := ArtifactSection{}
	opts := u.Sections["Artifact"]
	for _, opt := range opts {
		switch opt.Key {
		case "Artifact":
			a.Artifact = opt.Value
		case "AuthFile":
			a.AuthFile = opt.Value
		case "CertDir":
			a.CertDir = opt.Value
		case "ContainersConfModule":
			a.ContainersConfModule = append(a.ContainersConfModule, opt.Value)
		case "Creds":
			a.Creds = opt.Value
		case "DecryptionKey":
			a.DecryptionKey = opt.Value
		case "GlobalArgs":
			a.GlobalArgs = append(a.GlobalArgs, splitArgs(opt.Value)...)
		case "PodmanArgs":
			a.PodmanArgs = append(a.PodmanArgs, splitArgs(opt.Value)...)
		case "Quiet":
			a.Quiet = parseBool(opt.Value)
		case "Retry":
			if val, err := strconv.Atoi(opt.Value); err == nil {
				a.Retry = val
			}
		case "RetryDelay":
			a.RetryDelay = opt.Value
		case "ServiceName":
			a.ServiceName = opt.Value
		case "TLSVerify":
			a.TLSVerify = parseBool(opt.Value)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown key in [Artifact]: %s\n", opt.Key)
		}
	}
	return a
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
