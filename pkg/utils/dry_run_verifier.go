/*
Copyright 2014 The Kubernetes Authors.

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

package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
)

// DryRunVerifier verifies if a given group-version-kind supports DryRun
// against the current server. Sending dryRun requests to apiserver that
// don't support it will result in objects being unwillingly persisted.
//
// It reads the OpenAPI to see if the given GVK supports dryRun. If the
// GVK can not be found, we assume that CRDs will have the same level of
// support as "namespaces", and non-CRDs will not be supported. We
// delay the check for CRDs as much as possible though, since it
// requires an extra round-trip to the server.
type DryRunVerifier struct {
	Finder        cmdutil.CRDFinder
	OpenAPIGetter discovery.OpenAPISchemaInterface
}

// HasSupport verifies if the given gvk supports DryRun. An error is
// returned if it doesn't.
func (v *DryRunVerifier) HasSupport(gvk schema.GroupVersionKind) error {
	oapi, err := v.OpenAPIGetter.OpenAPISchema()
	if err != nil {
		return fmt.Errorf("failed to download openapi: %v", err)
	}
	supports, err := openapi.SupportsDryRun(oapi, gvk)
	if err != nil {
		// We assume that we couldn't find the type, then check for namespace:
		supports, _ = openapi.SupportsDryRun(oapi, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})
		// If namespace supports dryRun, then we will support dryRun for CRDs only.
		if supports {
			supports, err = v.Finder.HasCRD(gvk.GroupKind())
			if err != nil {
				return fmt.Errorf("failed to check CRD: %v", err)
			}
		}
	}
	if !supports {
		return fmt.Errorf("%v doesn't support dry-run", gvk)
	}
	return nil
}

// NewDryRunVerifier constructs a new DryRunVerifier
func NewDryRunVerifier(config *rest.Config) (*DryRunVerifier, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Dynamic Client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Discovery Client: %v", err)
	}

	return &DryRunVerifier{
		Finder:        cmdutil.NewCRDFinder(cmdutil.CRDFromDynamic(dynamicClient)),
		OpenAPIGetter: discoveryClient,
	}, nil
}
