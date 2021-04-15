package pipinstall

import (
	"os"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface InstallProcess --output fakes/install_process.go

// EntryResolver defines the interface for picking the most relevant entry from
// the Buildpack Plan entries.
type EntryResolver interface {
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

// InstallProcess defines the interface for installing the pip dependencies.
type InstallProcess interface {
	Execute(workingDir, targetDir, cacheDir string) error
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will install the pip dependencies by using the requirements.txt file
// to a packages layer. It also makes use of a cache layer to reuse the pip
// cache.
func Build(entryResolver EntryResolver, installProcess InstallProcess, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		packagesLayer, err := context.Layers.Get(PackagesLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		cacheLayer, err := context.Layers.Get(CacheLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		packagesLayer, err = packagesLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return installProcess.Execute(context.WorkingDir, packagesLayer.Path, cacheLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		packagesLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		packagesLayer.Launch, packagesLayer.Build = entryResolver.MergeLayerTypes(SitePackages, context.Plan.Entries)
		packagesLayer.Cache = packagesLayer.Build
		cacheLayer.Cache = true

		packagesLayer.SharedEnv.Default("PYTHONUSERBASE", packagesLayer.Path)
		logger.Process("Configuring environment")
		logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(packagesLayer.SharedEnv))
		logger.Break()

		layers := []packit.Layer{packagesLayer}
		if _, err := os.Stat(cacheLayer.Path); err == nil {
			if !fs.IsEmptyDir(cacheLayer.Path) {
				layers = append(layers, cacheLayer)
			}
		}

		result := packit.BuildResult{
			Layers: layers,
		}

		return result, nil
	}
}
