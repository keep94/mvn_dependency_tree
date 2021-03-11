package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	mvn "github.com/keep94/mvn_dependency_tree"
)

var (
	fTreeFile            string
	fDirectDependencyCsv string
	fLibraries           string
	fVersions            string
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

func buildDependencyRows(
	dependencies []dependencyType,
	libraries mvn.LibraryDB,
	versions mvn.VersionDB) []mvn.Dependency {
	var lastKey mvn.VersionKey
	var result []mvn.Dependency
	for _, d := range dependencies {
		var row mvn.Dependency
		row.Name = fmt.Sprintf("%s:%s", d.Group, d.Artifact)
		row.Version = d.Version
		if row.VersionKey == lastKey {
			continue
		}
		row.Date = versions.Date(row.Name, row.Version)
		library := libraries[row.Name]
		row.Latest = library.Latest
		row.LatestDate = versions.Date(row.Name, row.Latest)
		row.NewLocation = library.NewLocation
		row.Description = library.Description
		result = append(result, row)
		lastKey = row.VersionKey
	}
	return result
}

func scanDependencyFile(f io.Reader) ([]dependencyType, error) {
	scanner := bufio.NewScanner(f)
	dependencyScanner := newDependencyScannerType()
	for scanner.Scan() {
		dependencyScanner.Scan(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stringsToDependencies(dependencyScanner.Dependencies())
}

func main() {
	flag.Parse()
	var libraries []mvn.Library
	var versions []mvn.Version
	var err error
	if fLibraries != "" {
		libraries, err = mvn.ReadLibrariesFile(fLibraries)
		if err != nil {
			log.Fatal(err)
		}
	}
	if fVersions != "" {
		versions, err = mvn.ReadVersionsFile(fVersions)
		if err != nil {
			log.Fatal(err)
		}
	}
	libraryDB := mvn.NewLibraryDB(libraries)
	versionDB := mvn.NewVersionDB(versions)
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
	dependencies, err := scanDependencyFile(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := mvn.WriteDependencies(
		outf,
		buildDependencyRows(
			dependencies,
			libraryDB,
			versionDB)); err != nil {
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
		&fLibraries, "lib", "", "Library CSV File")
	flag.StringVar(
		&fVersions, "ver", "", "Version CSV File")
}
