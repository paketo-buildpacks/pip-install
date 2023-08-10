package pipinstall_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	pipinstall "github.com/paketo-buildpacks/pip-install"
	"github.com/paketo-buildpacks/pip-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		packagesLayerPath  string
		cacheLayerPath     string
		pipSourceLayerPath string
		workingDir         string
		executable         *fakes.Executable
		buffer             *bytes.Buffer

		pipInstallProcess pipinstall.PipInstallProcess
	)

	it.Before(func() {
		packagesLayerPath = t.TempDir()
		cacheLayerPath = t.TempDir()
		workingDir = t.TempDir()
		pipSourceLayerPath = t.TempDir()

		t.Setenv("PIP_FIND_LINKS", pipSourceLayerPath)

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			fmt.Fprintln(execution.Stdout, "stdout output")
			fmt.Fprintln(execution.Stderr, "stderr output")
			return nil
		}
		buffer = bytes.NewBuffer(nil)

		pipInstallProcess = pipinstall.NewPipInstallProcess(executable, scribe.NewEmitter(buffer))
	})

	context("Execute", func() {
		it("runs installation", func() {
			err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
				"Args": Equal([]string{
					"install",
					"--exists-action=w",
					fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
					"--compile",
					"--user",
					"--disable-pip-version-check",
					"--requirement=requirements.txt",
				}),
				"Dir": Equal(workingDir),
				"Env": ContainElements(
					fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
					fmt.Sprintf("PIP_FIND_LINKS=%s", pipSourceLayerPath),
				),
			}))
			Expect(buffer.String()).To(ContainLines(
				fmt.Sprintf("    Running 'pip install --exists-action=w --cache-dir=%s --compile --user --disable-pip-version-check --requirement=requirements.txt'", cacheLayerPath),
				"      stdout output",
				"      stderr output",
			))
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
						"--ignore-installed",
						"--exists-action=w",
						"--no-index",
						"--compile",
						"--user",
						"--disable-pip-version-check",
						"--requirement=requirements.txt",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElements(
						fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
						fmt.Sprintf("PIP_FIND_LINKS=%s", strings.Join([]string{pipSourceLayerPath, filepath.Join(workingDir, "vendor")}, " ")),
					),
				}))
				Expect(buffer.String()).To(ContainLines(
					"    Running 'pip install --ignore-installed --exists-action=w --no-index --compile --user --disable-pip-version-check --requirement=requirements.txt'",
					"      stdout output",
					"      stderr output",
				))
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
				t.Setenv("BP_PIP_DEST_PATH", "fake-vendor")
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
						"install",
						"--exists-action=w",
						fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
						"--compile",
						"--user",
						"--disable-pip-version-check",
						"--requirement=requirements.txt",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElements(
						fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
						fmt.Sprintf("PIP_FIND_LINKS=%s", pipSourceLayerPath),
					),
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
							"--ignore-installed",
							"--exists-action=w",
							"--no-index",
							"--compile",
							"--user",
							"--disable-pip-version-check",
							"--requirement=requirements.txt",
						}),
						"Dir": Equal(workingDir),
						"Env": ContainElements(
							fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
							fmt.Sprintf("PIP_FIND_LINKS=%s", strings.Join([]string{pipSourceLayerPath, filepath.Join(workingDir, "fake-vendor")}, " ")),
						),
					}))
				})
			})
		})

		context("when BP_PIP_REQUIREMENT overrides the default requirement path", func() {
			it.Before(func() {
				t.Setenv("BP_PIP_REQUIREMENT", "requirements-dev.txt")
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
						"install",
						"--exists-action=w",
						fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
						"--compile",
						"--user",
						"--disable-pip-version-check",
						"--requirement=requirements-dev.txt",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
				}))
			})
		})

		context("when BP_PIP_REQUIREMENT has multiple values", func() {
			it.Before(func() {
				t.Setenv("BP_PIP_REQUIREMENT", "requirements.txt requirements-lint.txt")
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
						"install",
						"--exists-action=w",
						fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
						"--compile",
						"--user",
						"--disable-pip-version-check",
						"--requirement=requirements.txt",
						"--requirement=requirements-lint.txt",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElement(fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath)),
				}))
			})
		})

		context("when BP_PIP_FIND_LINKS contains additional find-links dirs", func() {
			it.Before(func() {
				t.Setenv("BP_PIP_FIND_LINKS", "some-find-links-dir some-other-find-links-dir")
				t.Setenv("BP_PIP_DEST_PATH", "fake-vendor")
			})

			it("runs installation", func() {
				err := pipInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution).To(MatchFields(IgnoreExtras, Fields{
					"Args": Equal([]string{
						"install",
						"--exists-action=w",
						fmt.Sprintf("--cache-dir=%s", cacheLayerPath),
						"--compile",
						"--user",
						"--disable-pip-version-check",
						"--requirement=requirements.txt",
					}),
					"Dir": Equal(workingDir),
					"Env": ContainElements(
						fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
						fmt.Sprintf("PIP_FIND_LINKS=%s", strings.Join([]string{"some-find-links-dir", "some-other-find-links-dir", pipSourceLayerPath}, " ")),
					),
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
							"--ignore-installed",
							"--exists-action=w",
							"--no-index",
							"--compile",
							"--user",
							"--disable-pip-version-check",
							"--requirement=requirements.txt",
						}),
						"Dir": Equal(workingDir),
						"Env": ContainElements(
							fmt.Sprintf("PYTHONUSERBASE=%s", packagesLayerPath),
							fmt.Sprintf("PIP_FIND_LINKS=%s", strings.Join([]string{"some-find-links-dir", "some-other-find-links-dir", pipSourceLayerPath, filepath.Join(workingDir, "fake-vendor")}, " ")),
						),
					}))
				})
			})
		})
	})
}
