package remediation

type Package interface {
	Parser() (string, error)
}

type PackageContentJSON struct {
	FileContent       string
	PackageIdentifier string
	PackageVersion    string
}
