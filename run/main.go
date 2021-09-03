package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	pipinstall "github.com/paketo-buildpacks/pip-install"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout)

	packit.Run(
		pipinstall.Detect(),
		pipinstall.Build(
			draft.NewPlanner(),
			pipinstall.NewPipInstallProcess(pexec.NewExecutable("pip"), logger),
			pipinstall.NewSiteProcess(pexec.NewExecutable("python")),
			chronos.DefaultClock,
			logger,
		),
	)
}
