package pipinstall_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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

		installProcess      *fakes.InstallProcess
		sitePackagesProcess *fakes.SitePackagesProcess
		sbomGenerator       *fakes.SBOMGenerator

		buffer *bytes.Buffer

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		layersDir = t.TempDir()
		workingDir = t.TempDir()
		cnbDir = t.TempDir()

		installProcess = &fakes.InstallProcess{}
		sitePackagesProcess = &fakes.SitePackagesProcess{}
		sitePackagesProcess.ExecuteCall.Returns.SitePackagesPath = "some-site-packages-path"

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)

		build = pipinstall.Build(
			installProcess,
			sitePackagesProcess,
			sbomGenerator,
			chronos.DefaultClock,
			scribe.NewEmitter(buffer),
		)

		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name:     "site-packages",
						Metadata: map[string]interface{}{},
					},
				},
			},
			Platform: packit.Platform{Path: "some-platform-path"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		}
	})

	it("runs the build process and returns expected layers", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		layers := result.Layers
		Expect(layers).To(HaveLen(1))

		packagesLayer := layers[0]
		Expect(packagesLayer.Name).To(Equal("packages"))
		Expect(packagesLayer.Path).To(Equal(filepath.Join(layersDir, "packages")))

		Expect(packagesLayer.Build).To(BeFalse())
		Expect(packagesLayer.Launch).To(BeFalse())
		Expect(packagesLayer.Cache).To(BeFalse())

		Expect(packagesLayer.BuildEnv).To(BeEmpty())
		Expect(packagesLayer.LaunchEnv).To(BeEmpty())
		Expect(packagesLayer.ProcessLaunchEnv).To(BeEmpty())

		Expect(packagesLayer.SharedEnv).To(HaveLen(2))
		Expect(packagesLayer.SharedEnv["PYTHONPATH.prepend"]).To(Equal("some-site-packages-path"))
		Expect(packagesLayer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))

		Expect(packagesLayer.SBOM.Formats()).To(HaveLen(2))
		var actualExtensions []string
		for _, format := range packagesLayer.SBOM.Formats() {
			actualExtensions = append(actualExtensions, format.Extension)
		}
		Expect(actualExtensions).To(ConsistOf("cdx.json", "spdx.json"))

		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.TargetDir).To(Equal(filepath.Join(layersDir, "packages")))
		Expect(installProcess.ExecuteCall.Receives.CacheDir).To(Equal(filepath.Join(layersDir, "cache")))

		Expect(sitePackagesProcess.ExecuteCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "packages")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))

		Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(workingDir))
	})

	context("site-packages required at build and launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries[0].Metadata["launch"] = true
			buildContext.Plan.Entries[0].Metadata["build"] = true
		})

		it("layer's build, launch, cache flags must be set", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			packagesLayer := result.Layers[0]

			Expect(packagesLayer.Name).To(Equal("packages"))

			Expect(packagesLayer.Build).To(BeTrue())
			Expect(packagesLayer.Launch).To(BeTrue())
			Expect(packagesLayer.Cache).To(BeTrue())
		})
	})

	context("site-packages required at launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries[0].Metadata["launch"] = true
			buildContext.Plan.Entries[0].Metadata["build"] = false
		})

		it("layer's build, cache flags must be set", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			packagesLayer := result.Layers[0]

			Expect(packagesLayer.Name).To(Equal("packages"))

			Expect(packagesLayer.Build).To(BeFalse())
			Expect(packagesLayer.Launch).To(BeTrue())
			Expect(packagesLayer.Cache).To(BeTrue())
		})
	})

	context("install process utilizes cache", func() {
		it.Before(func() {
			installProcess.ExecuteCall.Stub = func(_, _, cachePath string) error {
				Expect(os.MkdirAll(filepath.Join(cachePath, "something"), os.ModePerm)).To(Succeed())
				return nil
			}
			buildContext.Plan.Entries[0].Metadata["launch"] = true
			buildContext.Plan.Entries[0].Metadata["build"] = true
		})

		it("result should include a cache layer", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(2))

			packagesLayer := result.Layers[0]
			Expect(packagesLayer.Name).To(Equal("packages"))
			Expect(packagesLayer.Path).To(Equal(filepath.Join(layersDir, "packages")))

			Expect(packagesLayer.Build).To(BeTrue())
			Expect(packagesLayer.Launch).To(BeTrue())
			Expect(packagesLayer.Cache).To(BeTrue())

			cacheLayer := result.Layers[1]
			Expect(cacheLayer.Name).To(Equal("cache"))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, "cache")))

			Expect(packagesLayer.Build).To(BeTrue())
			Expect(packagesLayer.Launch).To(BeTrue())
			Expect(cacheLayer.Cache).To(BeTrue())
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
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("could not run install process")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("could not run install process"))
			})
		})

		context("when site-packages process returns an error", func() {
			it.Before(func() {
				sitePackagesProcess.ExecuteCall.Returns.Err = errors.New("could not find site-packages path")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("could not find site-packages path"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})
	})
}
