package kicsengine

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	kicsassets "github.com/Checkmarx/kics/v2/assets"
	"github.com/Checkmarx/kics/v2/pkg/engine"
	"github.com/Checkmarx/kics/v2/pkg/engine/provider"
	"github.com/Checkmarx/kics/v2/pkg/engine/secrets"
	"github.com/Checkmarx/kics/v2/pkg/engine/source"
	"github.com/Checkmarx/kics/v2/pkg/kics"
	"github.com/Checkmarx/kics/v2/pkg/model"
	"github.com/Checkmarx/kics/v2/pkg/parser"
	ansibleConfigParser "github.com/Checkmarx/kics/v2/pkg/parser/ansible/ini/config"
	ansibleHostsParser "github.com/Checkmarx/kics/v2/pkg/parser/ansible/ini/hosts"
	bicepParser "github.com/Checkmarx/kics/v2/pkg/parser/bicep"
	buildahParser "github.com/Checkmarx/kics/v2/pkg/parser/buildah"
	dockerParser "github.com/Checkmarx/kics/v2/pkg/parser/docker"
	protoParser "github.com/Checkmarx/kics/v2/pkg/parser/grpc"
	jsonParser "github.com/Checkmarx/kics/v2/pkg/parser/json"
	terraformParser "github.com/Checkmarx/kics/v2/pkg/parser/terraform"
	yamlParser "github.com/Checkmarx/kics/v2/pkg/parser/yaml"
	"github.com/Checkmarx/kics/v2/pkg/progress"
	"github.com/Checkmarx/kics/v2/pkg/resolver"
	"github.com/Checkmarx/kics/v2/pkg/resolver/helm"
	kicsscanner "github.com/Checkmarx/kics/v2/pkg/scanner"
	"github.com/Checkmarx/kics/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

// These mirror the KICS CLI's own flag defaults, declared in
// internal/console/assets/scan-flags.json of the KICS module. The container engine invoked
// "kics scan" with every other flag left at its default, so matching them here is what keeps
// findings - and SimilarityIDs - identical between the two backends.
const (
	scanID            = "console"
	outputName        = "results"
	reportFormat      = "json"
	queryExecTimeout  = 60
	parallelWorkers   = 0
	maxFileSizeMB     = 5
	maxResolverDepth  = 15
	previewLines      = 3
	useOldSeverities  = false
	computeNewSimID   = false
	openAPIResolveRef = false
	disableSecrets    = false

	// ResultsFileName is the report the engine writes and iacrealtime reads back. Both
	// sides name it from here so the contract is a compile-time fact.
	ResultsFileName = outputName + "." + reportFormat

	resultsFilePerm = 0o600
	resultsDirPerm  = 0o700
)

// scanMu guards the working-directory swap in newInspector. os.Chdir is process-global while
// this mutex is only package-global, so it protects concurrent scans started through this
// package - it cannot protect against other code changing the working directory.
var scanMu sync.Mutex

// Options describes a single in-process KICS scan.
type Options struct {
	// ScanPath is the file or directory to scan.
	ScanPath string
	// OutputDir receives results.json, in the same layout the container engine produced.
	OutputDir string
	// SourceFile is the file being scanned. It narrows both the query set and the parser set
	// to what can actually match that file, which is the difference between a scan measured
	// in seconds and one measured in tens of seconds. Leave it empty to load every platform
	// and every parser, which is what the container engine did.
	SourceFile string
}

// Scan runs KICS in-process and writes results.json into opts.OutputDir, in the same shape
// the KICS container produced.
func Scan(ctx context.Context, opts Options) error {
	assetsRoot, err := AssetsRoot()
	if err != nil {
		return err
	}

	scanPath, err := filepath.Abs(opts.ScanPath)
	if err != nil {
		return errors.Wrap(err, "resolving scan path")
	}
	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return errors.Wrap(err, "resolving output path")
	}

	restoreLogging := silenceKicsLogging()
	defer restoreLogging()

	track := newTracker(previewLines)
	store := newMemoryStorage()

	services, err := buildServices(ctx, assetsRoot, scanPath, opts.SourceFile, track, store)
	if err != nil {
		return err
	}

	progressBar := progress.InitializePbBuilder(true, true, true)
	if scanErr := kicsscanner.PrepareAndScan(
		ctx, scanID, openAPIResolveRef, maxResolverDepth, *progressBar, services,
	); scanErr != nil {
		return errors.Wrap(scanErr, "running KICS scan")
	}

	vulnerabilities, err := store.GetVulnerabilities(ctx, scanID)
	if err != nil {
		return errors.Wrap(err, "collecting KICS results")
	}

	return writeReport(outputDir, scanPath, track, vulnerabilities)
}

func buildServices(
	ctx context.Context,
	assetsRoot, scanPath, sourceFile string,
	track *tracker,
	store kics.Storage,
) ([]*kics.Service, error) {
	platforms := PlatformsForFile(sourceFile)

	querySource := source.NewFilesystemSource(
		queryDirs(filepath.Join(assetsRoot, assetsDirName, queriesDirName), platforms),
		platforms,
		nil,
		filepath.Join(assetsRoot, assetsDirName, librariesDirName),
		false,
	)

	queryFilter := &source.QueryInspectorParameters{}

	// excludeResults carries no entries but must be non-nil: both inspectors index into it.
	excludeResults := map[string]bool{}

	inspector, err := newInspector(ctx, assetsRoot, querySource, queryFilter, track, excludeResults)
	if err != nil {
		return nil, err
	}

	secretsInspector, err := secrets.NewInspector(
		ctx, excludeResults, track, queryFilter, disableSecrets, queryExecTimeout,
		kicsassets.SecretsQueryRegexRulesJSON, false,
	)
	if err != nil {
		return nil, errors.Wrap(err, "initialising KICS secrets inspector")
	}

	filesSource, err := provider.NewFileSystemSourceProvider([]string{scanPath}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "opening scan path")
	}

	parsers, err := buildParsers(querySource, sourceFile)
	if err != nil {
		return nil, err
	}

	combinedResolver, err := resolver.NewBuilder().Add(&helm.Resolver{}).Build()
	if err != nil {
		return nil, errors.Wrap(err, "building KICS resolvers")
	}

	services := make([]*kics.Service, 0, len(parsers))
	for i := range parsers {
		services = append(services, &kics.Service{
			SourceProvider:   filesSource,
			Storage:          store,
			Parser:           parsers[i],
			Inspector:        inspector,
			SecretsInspector: secretsInspector,
			Tracker:          track,
			Resolver:         combinedResolver,
			MaxFileSize:      maxFileSizeMB,
		})
	}
	return services, nil
}

// newInspector builds the KICS inspector with the working directory pointed at the extracted
// assets, then restores it.
//
// engine.NewInspector reads ./assets/similarityID_transition relative to the executable
// directory and then the working directory, and neither is injectable in KICS v2. Without
// those files the engine silently skips similarity-ID transitions, which changes finding
// identity - the exact regression this backend must not introduce.
//
// The working directory is process-global, so the swap is held for this call only - the one
// KICS function that reads it - rather than for the whole scan, and scanMu keeps concurrent
// scans from overlapping inside the window. Every path handed to KICS elsewhere is absolute,
// so nothing else depends on the working directory.
func newInspector(
	ctx context.Context,
	assetsRoot string,
	querySource source.QueriesSource,
	queryFilter *source.QueryInspectorParameters,
	track *tracker,
	excludeResults map[string]bool,
) (inspector *engine.Inspector, err error) {
	scanMu.Lock()
	defer scanMu.Unlock()

	previousDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "reading working directory")
	}
	if chErr := os.Chdir(assetsRoot); chErr != nil {
		return nil, errors.Wrap(chErr, "entering KICS asset directory")
	}
	defer func() {
		if restoreErr := os.Chdir(previousDir); restoreErr != nil && err == nil {
			inspector, err = nil, errors.Wrap(restoreErr, "restoring working directory")
		}
	}()

	inspector, err = engine.NewInspector(
		ctx,
		querySource,
		engine.DefaultVulnerabilityBuilder,
		track,
		queryFilter,
		excludeResults,
		queryExecTimeout,
		useOldSeverities,
		false,
		parallelWorkers,
		computeNewSimID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "initialising KICS inspector")
	}
	return inspector, nil
}

// buildParsers mirrors the parser set the KICS CLI registers, so the same file types are
// recognised here as in the container.
//
// When the scanned file is known, parsers that cannot claim its extension are dropped. KICS
// runs every service it is given even when that service's parser matched no files, compiling
// and evaluating its whole query set against an empty payload - so a .tf scan would otherwise
// pay for the JSON parser's terraform queries too, because the JSON parser declares the
// terraform platform while only ever accepting .json files. The filter uses the same
// utils.GetExtension that KICS itself dispatches on, so it cannot disagree with the engine.
func buildParsers(querySource *source.FilesystemSource, sourceFile string) ([]*parser.Parser, error) {
	parsers, err := parser.NewBuilder().
		Add(&jsonParser.Parser{}).
		Add(&yamlParser.Parser{}).
		Add(terraformParser.NewDefault()).
		Add(&bicepParser.Parser{}).
		Add(&dockerParser.Parser{}).
		Add(&protoParser.Parser{}).
		Add(&buildahParser.Parser{}).
		Add(&ansibleConfigParser.Parser{}).
		Add(&ansibleHostsParser.Parser{}).
		Build(querySource.Types, querySource.CloudProviders)
	if err != nil {
		return nil, errors.Wrap(err, "building KICS parsers")
	}
	return parsersForFile(parsers, sourceFile), nil
}

// parsersForFile keeps only the parsers that could claim sourceFile. An unknown file, an
// unreadable one, or a filter that would leave nothing falls back to the full set, so the
// worst case is the old behaviour rather than a silent empty scan.
func parsersForFile(parsers []*parser.Parser, sourceFile string) []*parser.Parser {
	if sourceFile == "" {
		return parsers
	}
	extension, err := utils.GetExtension(sourceFile)
	if err != nil {
		return parsers
	}

	matching := make([]*parser.Parser, 0, len(parsers))
	for _, p := range parsers {
		if _, ok := p.SupportedExtensions()[extension]; ok {
			matching = append(matching, p)
		}
	}
	if len(matching) == 0 {
		return parsers
	}
	return matching
}

// writeReport emits the same results.json the container engine wrote, so every downstream
// consumer in the CLI keeps working unchanged.
func writeReport(outputDir, scanPath string, track *tracker, vulnerabilities []model.Vulnerability) error {
	counters := track.counters()
	summary := model.CreateSummary(&counters, vulnerabilities, scanID, nil, model.Version{})
	summary.ScannedPaths = []string{scanPath}
	now := time.Now()
	summary.Times = model.Times{Start: now, End: now}

	payload, err := json.MarshalIndent(summary, "", "\t")
	if err != nil {
		return errors.Wrap(err, "encoding KICS results")
	}

	if err := os.MkdirAll(outputDir, resultsDirPerm); err != nil {
		return errors.Wrap(err, "creating results directory")
	}

	target := filepath.Join(outputDir, ResultsFileName)
	if err := os.WriteFile(target, payload, resultsFilePerm); err != nil {
		return errors.Wrap(err, "writing KICS results")
	}
	return nil
}

// silenceKicsLogging redirects KICS's zerolog output for the duration of a scan. KICS logs
// verbosely to stderr by default, which would corrupt the CLI's own machine-readable output.
func silenceKicsLogging() func() {
	previousLogger := zerologlog.Logger
	previousLevel := zerolog.GlobalLevel()

	zerologlog.Logger = zerolog.New(io.Discard)
	zerolog.SetGlobalLevel(zerolog.Disabled)

	return func() {
		zerologlog.Logger = previousLogger
		zerolog.SetGlobalLevel(previousLevel)
	}
}
