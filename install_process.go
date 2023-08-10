package pipinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for invoking an executable.
type Executable interface {
	Execute(pexec.Execution) error
}

// PipInstallProcess implements the InstallProcess interface.
type PipInstallProcess struct {
	executable Executable
	logger     scribe.Emitter
}

// NewPipInstallProcess creates an instance of the PipInstallProcess given an Executable.
func NewPipInstallProcess(executable Executable, logger scribe.Emitter) PipInstallProcess {
	return PipInstallProcess{
		executable: executable,
		logger:     logger,
	}
}

// Execute installs the pip dependencies from workingDir/requirements.txt into
// the targetPath. The cachePath is used for the pip cache directory.
//
// The pip install command will install from local packages if they are found at
// the directory specified by `BP_PIP_DEST_PATH`, which defaults to `vendor`.
func (p PipInstallProcess) Execute(workingDir, targetPath, cachePath string) error {
	requirements, exists := os.LookupEnv("BP_PIP_REQUIREMENT")
	if !exists {
		requirements = "requirements.txt"
	}

	vendorDir := filepath.Join(workingDir, "vendor")
	if destPath, exists := os.LookupEnv("BP_PIP_DEST_PATH"); exists {
		vendorDir = filepath.Join(workingDir, destPath)
	}

	userFindLinks, _ := os.LookupEnv("BP_PIP_FIND_LINKS")
	findLinks, _ := os.LookupEnv("PIP_FIND_LINKS")

	combinedFindLinks := []string{userFindLinks, findLinks}

	var args []string
	if exists, err := fs.Exists(vendorDir); err != nil {
		return err
	} else if exists {
		combinedFindLinks = append(combinedFindLinks, vendorDir)
		args = offlineArgs(requirements)
	} else {
		args = onlineArgs(cachePath, requirements)
	}

	p.logger.Subprocess("Running 'pip %s'", strings.Join(args, " "))

	err := p.executable.Execute(pexec.Execution{
		Args: args,
		Env: append(os.Environ(),
			fmt.Sprintf("PYTHONUSERBASE=%s", targetPath),
			fmt.Sprintf("PIP_FIND_LINKS=%s", strings.TrimLeft(strings.Join(combinedFindLinks, " "), " ")),
		),
		Dir:    workingDir,
		Stdout: p.logger.ActionWriter,
		Stderr: p.logger.ActionWriter,
	})
	if err != nil {
		return fmt.Errorf("pip install failed:\nerror: %w", err)
	}

	return nil
}

func parseAppendArgs(key string, values string) []string {
	var rv []string
	for _, val := range strings.Split(values, " ") {
		rv = append(rv, fmt.Sprintf("--%s=%s", key, val))
	}
	return rv
}

func onlineArgs(cachePath string, requirements string) []string {
	rv := []string{
		"install",
		"--exists-action=w",
		fmt.Sprintf("--cache-dir=%s", cachePath),
		"--compile",
		"--user",
		"--disable-pip-version-check",
	}
	rv = append(rv, parseAppendArgs("requirement", requirements)...)
	return rv
}

func offlineArgs(requirements string) []string {
	rv := []string{
		"install",
		"--ignore-installed",
		"--exists-action=w",
		"--no-index",
		"--compile",
		"--user",
		"--disable-pip-version-check",
	}
	rv = append(rv, parseAppendArgs("requirement", requirements)...)
	return rv
}
