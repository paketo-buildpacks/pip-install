package pipinstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// the targetPath. The cachePath is used for the pip cache directory. If the
// vendor directory is present, the pip install command will install from local
// packages.
func (p PipInstallProcess) Execute(workingDir, targetPath, cachePath string) error {
	vendorDir := filepath.Join(workingDir, "vendor")

	var args []string
	_, err := os.Stat(vendorDir)
	if err == nil {
		args = offlineArgs(vendorDir)
	} else if errors.Is(err, os.ErrNotExist) {
		args = onlineArgs(cachePath)
	} else {
		return err
	}

	p.logger.Subprocess("Running 'pip %s'", strings.Join(args, " "))

	buffer := bytes.NewBuffer(nil)
	err = p.executable.Execute(pexec.Execution{
		Args:   args,
		Env:    append(os.Environ(), fmt.Sprintf("PYTHONUSERBASE=%s", targetPath)),
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return fmt.Errorf("pip install failed:\n%s\nerror: %w", buffer, err)
	}

	return nil
}

func onlineArgs(cachePath string) []string {
	return []string{
		"install",
		"--requirement",
		"requirements.txt",
		"--exists-action=w",
		fmt.Sprintf("--cache-dir=%s", cachePath),
		"--compile",
		"--user",
		"--disable-pip-version-check",
	}
}

func offlineArgs(vendorDir string) []string {
	return []string{
		"install",
		"--requirement",
		"requirements.txt",
		"--ignore-installed",
		"--exists-action=w",
		"--no-index",
		fmt.Sprintf("--find-links=%s", vendorDir),
		"--compile",
		"--user",
		"--disable-pip-version-check",
	}
}
