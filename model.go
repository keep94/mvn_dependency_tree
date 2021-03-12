package mvn_dependency_tree

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/jszwec/csvutil"
)

type Library struct {
	Name        string `csv:"name"`
	NewLocation string `csv:"new_location"`
	Latest      string `csv:"latest"`
	Description string `csv:"description"`
}

type VersionKey struct {
	Name    string `csv:"name"`
	Version string `csv:"version"`
}

type Version struct {
	VersionKey
	Date string `csv:"date"`
}

type Dependency struct {
	VersionKey
	Date        string `csv:"date"`
	Latest      string `csv:"latest"`
	LatestDate  string `csv:"latest_date"`
	NewLocation string `csv:"new_location"`
	Description string `csv:"description"`
}

type LibraryDB map[string]Library

func NewLibraryDB(libraries []Library) LibraryDB {
	result := make(LibraryDB)
	for _, library := range libraries {
		result[library.Name] = library
	}
	return result
}

func (db LibraryDB) Merge(d *Dependency) error {
	if d.Name == "" {
		return nil
	}
	library := db[d.Name]
	library.Name = d.Name
	if err := replace(
		d.Name, "latest", d.Latest, &library.Latest); err != nil {
		return err
	}
	if err := replace(
		d.Name,
		"new location",
		d.NewLocation,
		&library.NewLocation); err != nil {
		return err
	}
	if err := replace(
		d.Name,
		"description",
		d.Description,
		&library.Description); err != nil {
		return err
	}
	db[d.Name] = library
	return nil
}

func (db LibraryDB) Libraries() []Library {
	names := make([]string, 0, len(db))
	for name := range db {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Library, 0, len(db))
	for _, name := range names {
		result = append(result, db[name])
	}
	return result
}

type VersionDB map[VersionKey]Version

func NewVersionDB(versions []Version) VersionDB {
	result := make(VersionDB)
	for _, version := range versions {
		result[version.VersionKey] = version
	}
	return result
}

func (db VersionDB) Date(name, version string) string {
	result := db[VersionKey{name, version}]
	return result.Date
}

func (db VersionDB) Merge(d *Dependency) error {
	if err := db.mergeOne(d.VersionKey, d.Date); err != nil {
		return err
	}
	return db.mergeOne(VersionKey{d.Name, d.Latest}, d.LatestDate)
}

func (db VersionDB) mergeOne(key VersionKey, date string) error {
	if key.Name == "" || key.Version == "" {
		return nil
	}
	version := db[key]
	version.VersionKey = key
	keyName := fmt.Sprintf("%s+%s", key.Name, key.Version)
	if err := replace(keyName, "date", date, &version.Date); err != nil {
		return err
	}
	db[key] = version
	return nil
}

func (db VersionDB) Versions() []Version {
	keys := make([]VersionKey, 0, len(db))
	for key := range db {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Name < keys[j].Name {
			return true
		}
		if keys[i].Name > keys[j].Name {
			return false
		}
		return keys[i].Version < keys[j].Version
	})
	result := make([]Version, 0, len(db))
	for _, key := range keys {
		result = append(result, db[key])
	}
	return result
}

func ReadLibrariesFile(path string) ([]Library, error) {
	var result []Library
	if err := readCsvFile(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ReadVersionsFile(path string) ([]Version, error) {
	var result []Version
	if err := readCsvFile(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ReadDependenciesFile(path string) ([]Dependency, error) {
	var result []Dependency
	if err := readCsvFile(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func WriteLibrariesFile(path string, libraries []Library) error {
	return writeCsvFile(path, libraries)
}

func WriteVersionsFile(path string, versions []Version) error {
	return writeCsvFile(path, versions)
}

func WriteDependenciesFile(path string, dependencies []Dependency) error {
	return writeCsvFile(path, dependencies)
}

func WriteDependencies(w io.Writer, dependencies []Dependency) error {
	return writeCsv(w, dependencies)
}

type Merger interface {
	Merge(d *Dependency) error
}

func MergeAll(dependencies []Dependency, m Merger) error {
	for _, d := range dependencies {
		if err := m.Merge(&d); err != nil {
			return err
		}
	}
	return nil
}

func readCsvFile(path string, aSlicePtr interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return readCsv(f, aSlicePtr)
}

func readCsv(r io.Reader, aSlicePtr interface{}) error {
	decoder, err := csvutil.NewDecoder(csv.NewReader(r))
	if err != nil {
		return err
	}
	return decoder.Decode(aSlicePtr)
}

func writeCsvFile(path string, aSlice interface{}) error {
	f, err := os.OpenFile(
		path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCsv(f, aSlice)
}

func writeCsv(w io.Writer, aSlice interface{}) error {
	encoder := csvutil.NewEncoder(csv.NewWriter(w))
	return encoder.Encode(aSlice)
}

func replace(keyName, fieldName, value string, replace *string) error {
	if value == "" {
		return nil
	}
	if *replace == "" {
		*replace = value
		return nil
	}
	if *replace != value {
		return fmt.Errorf(
			"On '%s', had '%s' saw '%s' for '%s'",
			keyName, *replace, value, fieldName)
	}
	return nil
}
