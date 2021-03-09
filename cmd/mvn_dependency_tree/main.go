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

func writeDependencies(w io.Writer, dependencies []dependencyType) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Write([]string{
		"library", "version", "date", "latest", "latest_date"})
	if err := csvWriter.Error(); err != nil {
		return err
	}
	var csvLine [5]string
	var lastLibrary string
	var lastVersion string
	for _, d := range dependencies {
		csvLine[0] = fmt.Sprintf("%s:%s", d.Group, d.Artifact)
		csvLine[1] = d.Version
		if csvLine[0] == lastLibrary && csvLine[1] == lastVersion {
			continue
		}
		csvWriter.Write(csvLine[:])
		if err := csvWriter.Error(); err != nil {
			return err
		}
		lastLibrary = csvLine[0]
		lastVersion = csvLine[1]
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func main() {
	flag.Parse()
	var err error
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
	if err := writeDependencies(outf, dependencies); err != nil {
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
}
