package mvn_dependency_tree

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
)

type Library struct {
	Name        string
	NewLocation string
	Latest      string
	Description string
}

type libraryList []Library

func (s libraryList) Len() int { return len(s) }

func (_ libraryList) NumFields() int { return 4 }

func (_ libraryList) Title(fields []string) {
	fields[0] = "name"
	fields[1] = "new_location"
	fields[2] = "latest"
	fields[3] = "description"
}

func (s libraryList) Get(index int, fields []string) {
	fields[0] = s[index].Name
	fields[1] = s[index].NewLocation
	fields[2] = s[index].Latest
	fields[3] = s[index].Description
}

func (s *libraryList) Add(fields []string) {
	*s = append(*s, Library{
		Name:        fields[0],
		NewLocation: fields[1],
		Latest:      fields[2],
		Description: fields[3],
	})
}

type VersionKey struct {
	Name    string
	Version string
}

type Version struct {
	VersionKey
	Date string
}

type versionList []Version

func (s versionList) Len() int { return len(s) }

func (_ versionList) NumFields() int { return 3 }

func (_ versionList) Title(fields []string) {
	fields[0] = "name"
	fields[1] = "version"
	fields[2] = "date"
}

func (s versionList) Get(index int, fields []string) {
	fields[0] = s[index].Name
	fields[1] = s[index].Version
	fields[2] = s[index].Date
}

func (s *versionList) Add(fields []string) {
	*s = append(*s, Version{
		VersionKey: VersionKey{
			Name:    fields[0],
			Version: fields[1],
		},
		Date: fields[2],
	})
}

type Dependency struct {
	VersionKey
	Date        string
	Latest      string
	LatestDate  string
	NewLocation string
	Description string
}

type dependencyList []Dependency

func (s dependencyList) Len() int { return len(s) }

func (_ dependencyList) NumFields() int { return 7 }

func (_ dependencyList) Title(fields []string) {
	fields[0] = "name"
	fields[1] = "version"
	fields[2] = "date"
	fields[3] = "latest"
	fields[4] = "latest_date"
	fields[5] = "new_location"
	fields[6] = "description"
}

func (s dependencyList) Get(index int, fields []string) {
	fields[0] = s[index].Name
	fields[1] = s[index].Version
	fields[2] = s[index].Date
	fields[3] = s[index].Latest
	fields[4] = s[index].LatestDate
	fields[5] = s[index].NewLocation
	fields[6] = s[index].Description
}

func (s *dependencyList) Add(fields []string) {
	*s = append(*s, Dependency{
		VersionKey: VersionKey{
			Name:    fields[0],
			Version: fields[1],
		},
		Date:        fields[2],
		Latest:      fields[3],
		LatestDate:  fields[4],
		NewLocation: fields[5],
		Description: fields[6],
	})
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

func ReadLibraries(path string) ([]Library, error) {
	var result []Library
	if err := readCsvFile(path, (*libraryList)(&result)); err != nil {
		return nil, err
	}
	return result, nil
}

func ReadVersions(path string) ([]Version, error) {
	var result []Version
	if err := readCsvFile(path, (*versionList)(&result)); err != nil {
		return nil, err
	}
	return result, nil
}

func ReadDependencies(path string) ([]Dependency, error) {
	var result []Dependency
	if err := readCsvFile(path, (*dependencyList)(&result)); err != nil {
		return nil, err
	}
	return result, nil
}

func WriteLibraries(path string, libraries []Library) error {
	return writeCsvFile(path, libraryList(libraries))
}

func WriteVersions(path string, versions []Version) error {
	return writeCsvFile(path, versionList(versions))
}

func WriteDependencies(path string, dependencies []Dependency) error {
	return writeCsvFile(path, dependencyList(dependencies))
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

func readCsvFile(path string, consumer csvConsumer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return readCsv(f, consumer)
}

func readCsv(r io.Reader, consumer csvConsumer) error {
	csvReader := csv.NewReader(r)
	csvReader.ReuseRecord = true
	csvReader.FieldsPerRecord = consumer.NumFields()
	// Read title line
	_, err := csvReader.Read()
	if err != nil {
		return err
	}
	record, err := csvReader.Read()
	for err == nil {
		consumer.Add(record)
		record, err = csvReader.Read()
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func writeCsvFile(path string, producer csvProducer) error {
	f, err := os.OpenFile(
		path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCsv(f, producer)
}

func writeCsv(w io.Writer, producer csvProducer) error {
	csvWriter := csv.NewWriter(w)
	csvLine := make([]string, producer.NumFields())
	producer.Title(csvLine)
	csvWriter.Write(csvLine)
	if err := csvWriter.Error(); err != nil {
		return err
	}
	for i := 0; i < producer.Len(); i++ {
		producer.Get(i, csvLine)
		csvWriter.Write(csvLine)
		if err := csvWriter.Error(); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
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

type csvProducer interface {
	Len() int
	NumFields() int
	Title(fields []string)
	Get(index int, fields []string)
}

type csvConsumer interface {
	NumFields() int
	Add(fields []string)
}
