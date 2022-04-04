package pipinstall_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/scribe"
	pipinstall "github.com/paketo-buildpacks/pip-install"
	"github.com/paketo-buildpacks/pip-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

	it.After(func() {
		Expect(os.RemoveAll(packagesLayerPath)).To(Succeed())
		Expect(os.RemoveAll(cacheLayerPath)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.Unsetenv("BP_PIP_DEST_PATH")).NotTo(HaveOccurred())
	})

	context("Execute", func() {
		it("runs installation", func() {
			err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
				"Args": Equal([]string{
					"install",
					"--requirement",
					"requirements.txt",
					"--exists-action=w",
					fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
					"--compile",
					"--user",
					"--disable-pip-version-check",
				}),
				"Dir": Equal(workingDir),
				"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
			}))
		})

		context("when vendor directory exists", func() {
			it.Before(func() {
				Expect(os.Mkdir(filepath.Join(workingDir, "vendor"), os.ModeDir)).To(Succeed())
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
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
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
				}))
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

		context("when BP_PIP_DEST_PATH overrides the default vendor directory", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_PIP_DEST_PATH", "fake-vendor")).NotTo(HaveOccurred())
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
						"install",
						"--requirement",
						"requirements.txt",
						"--exists-action=w",
						fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
						"--compile",
						"--user",
						"--disable-pip-version-check",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
				}))
			})

			context("when vendor directory exists", func() {
				it.Before(func() {
					Expect(os.Mkdir(filepath.Join(workingDir, "fake-vendor"), os.ModeDir)).To(Succeed())
				})

				it("runs installation", func() {
					err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
					Expect(err).NotTo(HaveOccurred())

					Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
						"Args": Equal([]string{
							"install",
							"--requirement",
							"requirements.txt",
							"--ignore-installed",
							"--exists-action=w",
							"--no-index",
							fmt.Sprintf("--find-links=%s", filepath.Join(workingDir, "fake-vendor")),
							"--compile",
							"--user",
							"--disable-pip-version-check",
						}),
						"Dir": Equal(workingDir),
						"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
					}))
				})
			})
		})
	})
}
