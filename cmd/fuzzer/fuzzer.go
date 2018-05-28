/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/ingress-gce/cmd/fuzzer/app"
)

var (
	kubeconfig       *string
	ingressNamespace *string
	ingressName      *string
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "Subcommands: gen validate\n\n")
	}
	if len(os.Args) < 2 {
		fmt.Fprint(flag.CommandLine.Output(), "You need to specify a subcommand (one of: gen validate)\n")
		os.Exit(1)
	}

	// Make glog not complain about flags not being parsed.
	// flag.CommandLine.Parse([]string{})

	switch os.Args[1] {
	case "validate":
		app.ValidateFlagSet.Parse(os.Args[2:])
	default:
		flag.Usage()
		os.Exit(1)
	}

	// Suppress glog logging before flag.Parse() error.
	flag.CommandLine.Parse([]string{})

	switch os.Args[1] {
	case "validate":
		app.Validate()
	}
}
