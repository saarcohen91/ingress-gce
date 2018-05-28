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

package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/kr/pretty"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	backendconfig "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned"
	"k8s.io/ingress-gce/pkg/fuzz"
	"k8s.io/ingress-gce/pkg/fuzz/features"
	// Pull in the auth library for GCP.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	validateOptions struct {
		kubeconfig   string
		ns           string
		name         string
		listFeatures bool
		featureRegex string
	}
	// ValidateFlagSet is the flag set for the validate subcommand.
	ValidateFlagSet = flag.NewFlagSet("validate", flag.ExitOnError)
)

func init() {
	if home := homeDir(); home != "" {
		ValidateFlagSet.StringVar(&validateOptions.kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		ValidateFlagSet.StringVar(&validateOptions.kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	ValidateFlagSet.StringVar(&validateOptions.name, "name", "", "name of the Ingress object to validate")
	ValidateFlagSet.StringVar(&validateOptions.ns, "ns", "default", "namespace of the Ingress object to validate")
	ValidateFlagSet.BoolVar(&validateOptions.listFeatures, "listFeatures", false, "list features available to be validated")
	ValidateFlagSet.StringVar(&validateOptions.featureRegex, "featureRegex", "", "features matching regex will be included in validation")

	// Merges in the global flags into the subcommand FlagSet.
	flag.VisitAll(func(f *flag.Flag) {
		ValidateFlagSet.Var(f.Value, f.Name, f.Usage)
	})
}

// Validate the load balancer matches the Ingress spec.
func Validate() {
	if validateOptions.listFeatures {
		fmt.Println("Feature names:")
		for _, f := range features.All {
			fmt.Println(f.Name())
		}
		os.Exit(0)
	}

	if validateOptions.name == "" {
		fmt.Fprint(ValidateFlagSet.Output(), "You must specify a -name.\n")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", validateOptions.kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	k8s := k8sClientSet(config)
	env, err := fuzz.NewClientsetValidatorEnv(config, validateOptions.ns)

	if err != nil {
		panic(err)
	}

	var fs []fuzz.Feature
	if validateOptions.featureRegex == "" {
		fs = features.All
	} else {
		fregexp := regexp.MustCompile(validateOptions.featureRegex)
		for _, f := range features.All {
			if fregexp.Match([]byte(f.Name())) {
				fs = append(fs, f)
			}
		}
	}
	var fsNames []string
	for _, f := range fs {
		fsNames = append(fsNames, f.Name())
	}
	fmt.Printf("Features = %v\n\n", fsNames)

	ing, err := k8s.Extensions().Ingresses(validateOptions.ns).Get(validateOptions.name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Ingress =\n%s\n\n", pretty.Sprint(*ing))

	iv, err := fuzz.NewIngressValidator(env, ing, fs, nil)
	if err != nil {
		panic(err)
	}

	result := iv.Check(context.Background())
	fmt.Printf("Result =\n%s\n", pretty.Sprint(*result))

	if result.Err != nil {
		os.Exit(1)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func k8sClientSet(config *rest.Config) *kubernetes.Clientset {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func backendConfigClientset(config *rest.Config) *backendconfig.Clientset {
	clientset, err := backendconfig.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}
