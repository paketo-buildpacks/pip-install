package fakes

import "sync"

type SitePackagesProcess struct {
	ExecuteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			LayerPath string
		}
		Returns struct {
			SitePackagesPath string
			Err              error
		}
		Stub func(string) (string, error)
	}
}

func (f *SitePackagesProcess) Execute(param1 string) (string, error) {
	f.ExecuteCall.mutex.Lock()
	defer f.ExecuteCall.mutex.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.LayerPath = param1
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1)
	}
	return f.ExecuteCall.Returns.SitePackagesPath, f.ExecuteCall.Returns.Err
}
