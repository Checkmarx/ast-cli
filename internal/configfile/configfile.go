// Package configfile owns reading and writing the CLI's flat YAML
// configuration file. All mutations happen under a file lock so concurrent
// CLI processes cannot corrupt the file.
package configfile

import (
	"fmt"
	"io"
	"os"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

const (
	lockFileSuffix = ".lock"
	// ownerOnlyFilePerm keeps credential-bearing config files private on Unix;
	// Windows stores the equivalent in ACLs instead of POSIX modes.
	ownerOnlyFilePerm = 0o600
)

// Load reads the configuration map. A missing or empty file yields an empty
// map, which callers treat as "not configured yet".
func Load(path string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err := yaml.NewDecoder(file).Decode(&config); err != nil && err != io.EOF {
		// A genuinely corrupt file must error with a nil map so callers can
		// never mistake partial data for a valid config.
		return nil, fmt.Errorf("error decoding YAML: %w", err)
	}
	return config, nil
}

// Save writes the configuration map atomically: the content is written to a
// temporary sibling restricted to owner-only (0600) and renamed over the
// target. This keeps concurrent readers from seeing a truncated file and
// stops repeated writes from widening an existing file's permissions.
func Save(path string, config map[string]interface{}) error {
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, ownerOnlyFilePerm)
	if err != nil {
		return fmt.Errorf("opening config file for writing: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	encodeErr := encoder.Encode(config)
	closeErr := encoder.Close()
	closeFileErr := file.Close()
	if encodeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encoding YAML: %w", encodeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing config file: %w", closeErr)
	}
	if closeFileErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing config file: %w", closeFileErr)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing config file: %w", err)
	}
	return nil
}

// RemoveKey removes key from the config file under a file lock. Removing an
// absent key is a no-op.
func RemoveKey(path, key string) error {
	fileLock := flock.New(path + lockFileSuffix)
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("locking config file: %w", err)
	}
	if !locked {
		return fmt.Errorf("config file lock is held by another process")
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	config, err := Load(path)
	if err != nil {
		return err
	}
	if _, ok := config[key]; !ok {
		return nil
	}
	delete(config, key)
	return Save(path, config)
}

// SetKey assigns key to value in the config file under a file lock.
// SafeWriteSingleConfigKey-style helpers build on it.
func SetKey(path, key string, value interface{}) error {
	fileLock := flock.New(path + lockFileSuffix)
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("locking config file: %w", err)
	}
	if !locked {
		return fmt.Errorf("config file lock is held by another process")
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	config, err := Load(path)
	if err != nil {
		return err
	}
	config[key] = value
	return Save(path, config)
}
