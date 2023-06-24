package pipinstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

// BuildPlanMetadata is the buildpack specific data included in build plan
// requirements.
type BuildPlanMetadata struct {
	// Build denotes the dependency is needed at build-time.
	Build bool `toml:"build"`
}

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// Detection will contribute a Build Plan that provides site-packages,
// and requires cpython and pip at build.
func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		requirementPath, exists := os.LookupEnv("BP_PIP_REQUIREMENT")
		if !exists {
			requirementPath = "requirements.txt"
		}

		exists, err := fs.Exists(filepath.Join(context.WorkingDir, requirementPath))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if !exists {
			return packit.DetectResult{}, packit.Fail.WithMessage(fmt.Sprintf("no '%s' found", requirementPath))
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: SitePackages},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: CPython,
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: Pip,
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
				},
			},
		}, nil
	}
}
