package errorConstants

// HTTP Errors
const (
	StatusUnauthorized        = "you are not authorized to make this request"
	StatusForbidden           = "you are not allowed to make this request"
	StatusInternalServerError = "an error occurred during this request"
)

const (
	ApplicationDoesntExistOrNoPermission = "provided application does not exist or user has no permission to the application"
	ImportFilePathIsRequired             = "importFilePath is required"
	ProjectNameIsRequired                = "project name is required"
	FailedToGetApplication               = "failed to get application"
	SarifInvalidFileExtension            = "Invalid file extension. Supported extensions are .sarif and .zip containing sarif files."
)
