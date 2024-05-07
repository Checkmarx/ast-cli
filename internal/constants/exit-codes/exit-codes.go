package exitcodes

const (
	MultipleEnginesFailedExitCode   = 1
	SastEngineFailedExitCode        = 2
	ScaEngineFailedExitCode         = 3
	IacSecurityEngineFailedExitCode = 4 // Same code as kics to support forward compatibility
	KicsEngineFailedExitCode        = 4
	ApisecEngineFailedExitCode      = 5
)
