package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	pipinstall "github.com/paketo-community/pip-install"
)

func main() {
	planner := draft.NewPlanner()
	logger := scribe.NewEmitter(os.Stdout)
	installProcess := pipinstall.NewPipInstallProcess(pexec.NewExecutable("pip"), logger)

	packit.Run(
		pipinstall.Detect(),
		pipinstall.Build(
			planner,
			installProcess,
			chronos.DefaultClock,
			logger,
		),
	)
}
