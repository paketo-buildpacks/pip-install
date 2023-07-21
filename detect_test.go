package pipinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	pipinstall "github.com/paketo-buildpacks/pip-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		detect     packit.DetectFunc
		workingDir string
	)

	it.Before(func() {
		workingDir = t.TempDir()

		err := os.WriteFile(filepath.Join(workingDir, "requirements.txt"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		detect = pipinstall.Detect()
	})

	context("detection", func() {
		it("returns a build plan that provides site packages", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: pipinstall.SitePackages},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: pipinstall.CPython,
						Metadata: pipinstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: pipinstall.Pip,
						Metadata: pipinstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})

		context("when there is no requirements.txt file", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workingDir, "requirements.txt"))).To(Succeed())
			})

			it("fails detection", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(packit.Fail.WithMessage("BP_PIP_REQUIREMENT not set and no 'requirements.txt' found")))
			})
		})

		context("failure cases", func() {
			context("when the requirements.txt cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})

	})
}
