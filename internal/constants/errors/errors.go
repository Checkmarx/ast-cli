package errorconstants

import "fmt"

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
	NoASCALicense                          = "User doesn't have \"AI Protection\" or \"Checkmarx One Assist\" license"
	FailedUploadFileMsgWithDomain          = "Unable to upload the file to the pre-signed URL. Try adding the domain: %s to your allow list."
	FailedUploadFileMsgWithURL             = "Unable to upload the file to the pre-signed URL. Try adding the URL: %s to your allow list."

	// asca Engine
	FileExtensionIsRequired = "file must have an extension"

	// Realtime
	RealtimeEngineErrFormat        = "realtime engine error: %s"
	RealtimeEngineNotAvailable     = "Realtime engine is not available for this tenant"
	RealtimeEngineFilePathRequired = "file path is required for realtime scan"
)

type RealtimeEngineError struct {
	Message string
}

func (e *RealtimeEngineError) Error() error {
	return fmt.Errorf(RealtimeEngineErrFormat, e.Message)
}

func NewRealtimeEngineError(message string) *RealtimeEngineError {
	return &RealtimeEngineError{
		Message: message,
	}
}
