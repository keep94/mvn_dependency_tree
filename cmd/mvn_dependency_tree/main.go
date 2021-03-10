package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

var (
	fTreeFile            string
	fDirectDependencyCsv string
	fStore               string
)

type entryType struct {
	name  string
	level int
}

func toEntryType(line string) *entryType {
	if !strings.HasPrefix(line, "[INFO] ") {
		return nil
	}
	line = line[7:]
	level := 1
	for strings.HasPrefix(line, "|  ") || strings.HasPrefix(line, "   ") {
		level++
		line = line[3:]
	}
	if !strings.HasPrefix(line, "+- ") && !strings.HasPrefix(line, "\\- ") {
		return nil
	}
	line = line[3:]
	return &entryType{name: line, level: level}
}

type dependencyScannerType struct {
	scanLevel int
	names     map[string]struct{}
}

func newDependencyScannerType() *dependencyScannerType {
	return &dependencyScannerType{
		scanLevel: 1, names: make(map[string]struct{})}
}

func (d *dependencyScannerType) Scan(line string) {
	entry := toEntryType(line)
	if entry == nil {
		return
	}
	if entry.level > d.scanLevel {
		return
	}
	d.scanLevel = entry.level
	if strings.Contains(entry.name, "com.sunnylabs") || strings.Contains(entry.name, "com.wavefront") {
		d.scanLevel++
		return
	}
	d.names[entry.name] = struct{}{}
}

func (d *dependencyScannerType) Dependencies() []string {
	result := make([]string, 0, len(d.names))
	for n := range d.names {
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}

type dependencyType struct {
	Group    string
	Artifact string
	Format   string
	Version  string
	Target   string
}

func newDependency(str string) (result dependencyType, err error) {
	fields := strings.SplitN(str, ":", 5)
	if len(fields) < 5 {
		err = fmt.Errorf("'%s' lacks all 5 dependency fields", str)
		return
	}
	return dependencyType{
		Group:    fields[0],
		Artifact: fields[1],
		Format:   fields[2],
		Version:  fields[3],
		Target:   fields[4],
	}, nil
}

func stringsToDependencies(strs []string) ([]dependencyType, error) {
	result := make([]dependencyType, 0, len(strs))
	for _, str := range strs {
		dependency, err := newDependency(str)
		if err != nil {
			return nil, err
		}
		result = append(result, dependency)
	}
	return result, nil
}

type dependencyRowType struct {
	Name       string
	Version    string
	Date       string
	Latest     string
	LatestDate string
}

func readDependencies(r io.Reader) ([]dependencyRowType, error) {
	csvReader := csv.NewReader(r)
	csvReader.ReuseRecord = true
	// Read title line
	_, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	var result []dependencyRowType
	record, err := csvReader.Read()
	for err == nil {
		result = append(
			result,
			dependencyRowType{
				Name:       record[0],
				Version:    record[1],
				Date:       record[2],
				Latest:     record[3],
				LatestDate: record[4],
			})
		record, err = csvReader.Read()
	}
	if err == io.EOF {
		return result, nil
	}
	return nil, err
}

type nameVersionType struct {
	Name    string
	Version string
}

type versionStoreType struct {
	dates          map[nameVersionType]string
	latestVersions map[string]string
}

func buildVersionStore(rows []dependencyRowType) *versionStoreType {
	result := &versionStoreType{
		dates:          make(map[nameVersionType]string),
		latestVersions: make(map[string]string)}
	for _, row := range rows {
		if row.Name == "" {
			continue
		}
		if row.Version != "" && row.Date != "" {
			result.dates[nameVersionType{row.Name, row.Version}] = row.Date
		}
		if row.Latest != "" && row.LatestDate != "" {
			result.dates[nameVersionType{row.Name, row.Latest}] = row.LatestDate
		}
		if row.Latest != "" {
			result.latestVersions[row.Name] = row.Latest
		}
	}
	return result
}

func (s *versionStoreType) Date(name, version string) string {
	return s.dates[nameVersionType{name, version}]
}

func (s *versionStoreType) LatestVersion(name string) string {
	return s.latestVersions[name]
}

func readVersionStore(path string) (
	result *versionStoreType, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	rows, err := readDependencies(f)
	if err != nil {
		return
	}
	result = buildVersionStore(rows)
	return
}

func buildDependencyRows(
	dependencies []dependencyType,
	store *versionStoreType) []dependencyRowType {
	var lastName string
	var lastVersion string
	var result []dependencyRowType
	for _, d := range dependencies {
		var row dependencyRowType
		row.Name = fmt.Sprintf("%s:%s", d.Group, d.Artifact)
		row.Version = d.Version
		if row.Name == lastName && row.Version == lastVersion {
			continue
		}
		if store != nil {
			row.Date = store.Date(row.Name, row.Version)
			row.Latest = store.LatestVersion(row.Name)
			row.LatestDate = store.Date(row.Name, row.Latest)
		}
		result = append(result, row)
		lastName = row.Name
		lastVersion = row.Version
	}
	return result
}

func writeDependencies(w io.Writer, rows []dependencyRowType) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Write([]string{
		"library", "version", "date", "latest", "latest_date"})
	if err := csvWriter.Error(); err != nil {
		return err
	}
	var csvLine [5]string
	for _, row := range rows {
		csvLine[0] = row.Name
		csvLine[1] = row.Version
		csvLine[2] = row.Date
		csvLine[3] = row.Latest
		csvLine[4] = row.LatestDate
		csvWriter.Write(csvLine[:])
		if err := csvWriter.Error(); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func main() {
	flag.Parse()
	var store *versionStoreType
	var err error
	if fStore != "" {
		store, err = readVersionStore(fStore)
		if err != nil {
			log.Fatal(err)
		}
	}
	f := os.Stdin
	if fTreeFile != "" {
		f, err = os.Open(fTreeFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}
	outf := os.Stdout
	if fDirectDependencyCsv != "" {
		outf, err = os.OpenFile(
			fDirectDependencyCsv, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
		defer outf.Close()
	}
	scanner := bufio.NewScanner(f)
	dependencyScanner := newDependencyScannerType()
	for scanner.Scan() {
		dependencyScanner.Scan(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	dependencies, err := stringsToDependencies(
		dependencyScanner.Dependencies())
	if err != nil {
		log.Fatal(err)
	}
	if err := writeDependencies(
		outf, buildDependencyRows(dependencies, store)); err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(
		&fTreeFile,
		"tree",
		"",
		"mvn dependency:tree file. Empty means stdin")
	flag.StringVar(
		&fDirectDependencyCsv, "csv", "", "Direct dependency csv file")
	flag.StringVar(
		&fStore, "store", "", "Previous CSV file with versions")
}
