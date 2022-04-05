package fakes

import "sync"

type InstallProcess struct {
	ExecuteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			TargetDir  string
			CacheDir   string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
}

func (f *InstallProcess) Execute(param1 string, param2 string, param3 string) error {
	f.ExecuteCall.mutex.Lock()
	defer f.ExecuteCall.mutex.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	f.ExecuteCall.Receives.TargetDir = param2
	f.ExecuteCall.Receives.CacheDir = param3
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3)
	}
	return f.ExecuteCall.Returns.Error
}
