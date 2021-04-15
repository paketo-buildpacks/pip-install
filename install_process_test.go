package pipinstall_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/scribe"
	pipinstall "github.com/paketo-community/pip-install"
	"github.com/paketo-community/pip-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		packagesLayerPath string
		cacheLayerPath    string
		workingDir        string
		executable        *fakes.Executable

		pipInstallProcess pipinstall.PipInstallProcess
	)

	it.Before(func() {
		var err error
		packagesLayerPath, err = ioutil.TempDir("", "packages")
		Expect(err).NotTo(HaveOccurred())

		cacheLayerPath, err = ioutil.TempDir("", "cache")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "workingdir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}

		pipInstallProcess = pipinstall.NewPipInstallProcess(executable, scribe.NewEmitter(bytes.NewBuffer(nil)))
	})

	context("Execute", func() {
		it("runs installation", func() {
			err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
				"install",
				"--requirement",
				"requirements.txt",
				"--ignore-installed",
				"--exists-action=w",
				fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
				"--compile",
				"--user",
				"--disable-pip-version-check",
			}))
			Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workingDir))
			Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)))
		})

		context("when vendor directory exists", func() {
			it.Before(func() {
				Expect(os.Mkdir(filepath.Join(workingDir, "vendor"), os.ModeDir)).To(Succeed())
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
					"install",
					"--requirement",
					"requirements.txt",
					"--ignore-installed",
					"--exists-action=w",
					"--no-index",
					fmt.Sprintf("--find-links=%s", filepath.Join(workingDir, "vendor")),
					"--compile",
					"--user",
					"--disable-pip-version-check",
				}))
				Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workingDir))
				Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)))
			})
		})

		context("failure cases", func() {
			context("when vendor stat fails", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
