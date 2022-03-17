package pipinstall

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
)

// SiteProcess implements the Executable interface.
type SiteProcess struct {
	executable Executable
}

// NewSiteProcess creates an instance of the SiteProcess given an Executable that runs `python`
func NewSiteProcess(executable Executable) SiteProcess {
	return SiteProcess{
		executable: executable,
	}
}

// Execute runs a python command to locate the site packages within the pip targetLayerPath.
func (p SiteProcess) Execute(layerPath string) (string, error) {
	buffer := bytes.NewBuffer(nil)

	err := p.executable.Execute(pexec.Execution{
		Args:   []string{"-m", "site", "--user-site"},
		Env:    append(os.Environ(), fmt.Sprintf("PYTHONUSERBASE=%s", layerPath)),
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return "", fmt.Errorf("failed to locate site packages:\n%s\nerror: %w", buffer.String(), err)
	}

	path := strings.TrimSpace(buffer.String())

	if len(path) == 0 {
		return "", fmt.Errorf("failed to locate site packages: output is empty")
	}

	return path, nil
}
