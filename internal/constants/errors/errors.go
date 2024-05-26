package errorconstants

// HTTP Errors
const (
	StatusUnauthorized        = "you are not authorized to make this request"
	StatusForbidden           = "you are not allowed to make this request"
	RedirectURLNotFound       = "redirect URL not found in response"
	HTTPMethodNotFound        = "HTTP method not found in request"
	StatusInternalServerError = "an error occurred during this request"
)

const (
	ApplicationDoesntExistOrNoPermission   = "provided application does not exist or user has no permission to the application"
	ImportFilePathIsRequired               = "importFilePath is required"
	ProjectNameIsRequired                  = "project name is required"
	ProjectNotExists                       = "the project name you provided does not match any project"
	ScanIDRequired                         = "scan ID is required"
	FailedToGetApplication                 = "failed to get application"
	SarifInvalidFileExtension              = "Invalid file extension. Supported extensions are .sarif and .zip containing sarif files."
	ImportSarifFileError                   = "There was a problem importing the SARIF file. Please contact support for further details."
	ImportSarifFileErrorMessageWithMessage = "There was a problem importing the SARIF file. Please contact support for further details with the following error code: %d %s"

	// Lightweight Engine
	SourceCodeIsRequired    = "source code is required"
	EngineVersionIsRequired = "engine version is required"
	FileExtensionIsRequired = "file must have an extension"
)
