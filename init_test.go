package pipinstall_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPipInstall(t *testing.T) {
	suite := spec.New("pipinstall", spec.Report(report.Terminal{}))
	suite("Detect", testDetect)
	suite("Build", testBuild)
	suite("InstallProcess", testInstallProcess)
	suite("SiteProcess", testSiteProcess)
	suite.Run(t)
}
