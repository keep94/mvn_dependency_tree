mvn_dependency_tree
===================

This program reads the output of mvn dependency:tree and creates a CSV file
containing all the Wavefront direct dependencies.

Sample usage:

```sh
$ mvn dependency:tree | ~/go/bin/mvn_dependency_tree > direct_dependencies.csv
```
