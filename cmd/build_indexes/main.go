package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	mvn "github.com/keep94/mvn_dependency_tree"
)

var (
	fLibraryIn  string
	fVersionIn  string
	fLibraryOut string
	fVersionOut string
)

func main() {
	flag.Parse()
	if fLibraryOut == "" || fVersionOut == "" {
		fmt.Println("Both lout and vout flags are required.")
		flag.Usage()
		os.Exit(2)
	}
	var err error
	var libraries []mvn.Library
	if fLibraryIn != "" {
		libraries, err = mvn.ReadLibrariesFile(fLibraryIn)
		if err != nil {
			log.Fatal(err)
		}
	}
	var versions []mvn.Version
	if fVersionIn != "" {
		versions, err = mvn.ReadVersionsFile(fVersionIn)
		if err != nil {
			log.Fatal(err)
		}
	}
	libraryDB := mvn.NewLibraryDB(libraries)
	versionDB := mvn.NewVersionDB(versions)
	for _, dependencyFile := range flag.Args() {
		dependencies, err := mvn.ReadDependenciesFile(dependencyFile)
		if err != nil {
			log.Fatal(err)
		}
		if err := mvn.MergeAll(dependencies, libraryDB); err != nil {
			log.Fatalf("Error processing %s: %v", dependencyFile, err)
		}
		if err := mvn.MergeAll(dependencies, versionDB); err != nil {
			log.Fatalf("Error processing %s: %v", dependencyFile, err)
		}
	}
	err = mvn.WriteLibrariesFile(fLibraryOut, libraryDB.Libraries())
	if err != nil {
		log.Fatal(err)
	}
	err = mvn.WriteVersionsFile(fVersionOut, versionDB.Versions())
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(
		&fLibraryIn, "lin", "", "Input library CSV")
	flag.StringVar(
		&fVersionIn, "vin", "", "Input version CSV")
	flag.StringVar(
		&fLibraryOut, "lout", "", "Output library CSV")
	flag.StringVar(
		&fVersionOut, "vout", "", "Output version CSV")
}
