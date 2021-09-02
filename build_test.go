package pipinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
	pipinstall "github.com/paketo-buildpacks/pip-install"
	"github.com/paketo-buildpacks/pip-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		clock          chronos.Clock
		timeStamp      time.Time
		installProcess *fakes.InstallProcess
		buffer         *bytes.Buffer
		logEmitter     scribe.Emitter
		entryResolver  *fakes.EntryResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		installProcess = &fakes.InstallProcess{}
		entryResolver = &fakes.EntryResolver{}

		buffer = bytes.NewBuffer(nil)
		logEmitter = scribe.NewEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		build = pipinstall.Build(entryResolver, installProcess, clock, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("runs the build process and returns expected layers", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: pipinstall.SitePackages,
					},
				},
			},
			Platform: packit.Platform{Path: "some-platform-path"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name: pipinstall.PackagesLayerName,
					Path: filepath.Join(layersDir, pipinstall.PackagesLayerName),
					SharedEnv: packit.Environment{
						"PYTHONUSERBASE.default": filepath.Join(layersDir, pipinstall.PackagesLayerName),
					},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
					Metadata: map[string]interface{}{
						"built_at": timeStamp.Format(time.RFC3339Nano),
						//"cache_sha": "",
					},
				},
			},
		}))

		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.TargetDir).To(Equal(filepath.Join(layersDir, pipinstall.PackagesLayerName)))
		Expect(installProcess.ExecuteCall.Receives.CacheDir).To(Equal(filepath.Join(layersDir, pipinstall.CacheLayerName)))

		Expect(entryResolver.MergeLayerTypesCall.Receives.Name).To(Equal(pipinstall.SitePackages))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: pipinstall.SitePackages},
		}))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
	})

	context("site-packages required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("layer's build, launch, cache flags must be set", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: pipinstall.SitePackages,
						},
					},
				},
				Platform: packit.Platform{Path: "some-platform-path"},
				Layers:   packit.Layers{Path: layersDir},
				Stack:    "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name: pipinstall.PackagesLayerName,
						Path: filepath.Join(layersDir, pipinstall.PackagesLayerName),
						SharedEnv: packit.Environment{
							"PYTHONUSERBASE.default": filepath.Join(layersDir, pipinstall.PackagesLayerName),
						},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at": timeStamp.Format(time.RFC3339Nano),
							//"cache_sha": "",
						},
					},
				},
			}))
		})
	})

	context("site-packages required at launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = false
		})

		it("layer's build, cache flags must be set", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: pipinstall.SitePackages,
						},
					},
				},
				Platform: packit.Platform{Path: "some-platform-path"},
				Layers:   packit.Layers{Path: layersDir},
				Stack:    "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name: pipinstall.PackagesLayerName,
						Path: filepath.Join(layersDir, pipinstall.PackagesLayerName),
						SharedEnv: packit.Environment{
							"PYTHONUSERBASE.default": filepath.Join(layersDir, pipinstall.PackagesLayerName),
						},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            false,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at": timeStamp.Format(time.RFC3339Nano),
						},
					},
				},
			}))
		})
	})

	context("install process utilizes cache", func() {
		it.Before(func() {
			installProcess.ExecuteCall.Stub = func(_, _, cachePath string) error {
				err := os.MkdirAll(filepath.Join(cachePath, "something"), os.ModePerm)
				if err != nil {
					return fmt.Errorf("issue with stub call: %+v", err)
				}
				return nil
			}
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("result should include a cache layer", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: pipinstall.SitePackages,
						},
					},
				},
				Platform: packit.Platform{Path: "some-platform-path"},
				Layers:   packit.Layers{Path: layersDir},
				Stack:    "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name: pipinstall.PackagesLayerName,
						Path: filepath.Join(layersDir, pipinstall.PackagesLayerName),
						SharedEnv: packit.Environment{
							"PYTHONUSERBASE.default": filepath.Join(layersDir, pipinstall.PackagesLayerName),
						},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at": timeStamp.Format(time.RFC3339Nano),
							//"cache_sha": "",
						},
					},
					{
						Name:             pipinstall.CacheLayerName,
						Path:             filepath.Join(layersDir, pipinstall.CacheLayerName),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            false,
						Launch:           false,
						Cache:            true,
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the layers directory cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "site-packages",
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "site-packages",
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("some-error")))
			})
		})
	})
}
