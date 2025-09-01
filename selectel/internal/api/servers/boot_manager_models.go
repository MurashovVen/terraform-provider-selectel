package servers

type (
	LocalDrives map[string]*LocalDrive

	LocalDrive struct {
		Type  string           `json:"type"`
		Match *LocalDriveMatch `json:"match"`
	}

	LocalDriveMatch struct {
		Size int    `json:"size"`
		Type string `json:"type"`
	}
)

// todo ask 1 ok?
func (ld LocalDrives) GetMaxPresentedType() string {
	var (
		maxTypeCount = 0
		maxType      = ""
		typeCount    = make(map[string]int)
	)
	for _, ldd := range ld {
		cc := typeCount[ldd.Match.Type]
		cc++

		if cc > maxTypeCount {
			maxTypeCount = cc
			maxType = ldd.Match.Type
		}
	}

	return maxType
}

type OperatingSystem struct {
	UUID              string                 `json:"uuid"`
	Name              string                 `json:"os_name"`
	OSValue           string                 `json:"os_value"`
	Arch              string                 `json:"arch"`
	VersionValue      string                 `json:"version_value"`
	ScriptAllowed     bool                   `json:"script_allowed"`
	IsSSHKeyAllowed   bool                   `json:"is_ssh_key_allowed"`
	Partitioning      bool                   `json:"partitioning"`
	TemplateVersion   string                 `json:"template_version"`
	DefaultPartitions []*PartitionConfigItem `json:"default_partitions"`
}

func (os *OperatingSystem) IsPrivateNetworkAvailable() bool {
	return os.OSValue != "windows" && os.TemplateVersion == "v2"
}

type OperatingSystems []*OperatingSystem

func (o OperatingSystems) FindOneByNameAndVersion(name, version string) *OperatingSystem {
	for _, os := range o {
		if os.Name == name && os.VersionValue == version {
			return os
		}
	}

	return nil
}

func (o OperatingSystems) FindOneByID(id string) *OperatingSystem {
	for _, os := range o {
		if os.UUID == id {
			return os
		}
	}

	return nil
}

func (o OperatingSystems) FindOneByArchAndVersionAndOs(arch, version, osValue string) *OperatingSystem {
	for _, os := range o {
		if os.Arch == arch && os.VersionValue == version && os.OSValue == osValue {
			return os
		}
	}

	return nil
}

type OperatingSystemAtResource struct {
	UserSSHKey   string `json:"user_ssh_key"`
	UserHostName string `json:"userhostname"`
	UserScript   string `json:"user_script"`
	Password     string `json:"password"`
	OSValue      string `json:"os_template"`
	Arch         string `json:"arch"`
	Version      string `json:"version"`
	Reinstall    int    `json:"reinstall"`
}
