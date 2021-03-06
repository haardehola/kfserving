/*
Copyright 2020 kubeflow.org.

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

package v1beta1

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/kfserving/pkg/constants"
	"github.com/kubeflow/kfserving/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlibiExplainerType is the explanation method
type AlibiExplainerType string

// AlibiExplainerType Enum
const (
	AlibiAnchorsTabularExplainer  AlibiExplainerType = "AnchorTabular"
	AlibiAnchorsImageExplainer    AlibiExplainerType = "AnchorImages"
	AlibiAnchorsTextExplainer     AlibiExplainerType = "AnchorText"
	AlibiCounterfactualsExplainer AlibiExplainerType = "Counterfactuals"
	AlibiContrastiveExplainer     AlibiExplainerType = "Contrastive"
)

// AlibiExplainerSpec defines the arguments for configuring an Alibi Explanation Server
type AlibiExplainerSpec struct {
	// The type of Alibi explainer <br />
	// Valid values are: <br />
	// - "AnchorTabular"; <br />
	// - "AnchorImages"; <br />
	// - "AnchorText"; <br />
	// - "Counterfactuals"; <br />
	// - "Contrastive"; <br />
	Type AlibiExplainerType `json:"type"`
	// The location of a trained explanation model
	// +optional
	StorageURI string `json:"storageUri,omitempty"`
	// Alibi docker image version, defaults to latest Alibi Version
	// +optional
	RuntimeVersion *string `json:"runtimeVersion,omitempty"`
	// Inline custom parameter settings for explainer
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// Container enables overrides for the predictor.
	// Each framework will have different defaults that are populated in the underlying container spec.
	// +optional
	v1.Container `json:",inline"`
}

var _ ComponentImplementation = &AlibiExplainerSpec{}

func (s *AlibiExplainerSpec) GetStorageUri() *string {
	return &s.StorageURI
}

func (s *AlibiExplainerSpec) GetResourceRequirements() *v1.ResourceRequirements {
	// return the ResourceRequirements value if set on the spec
	return &s.Resources
}

func (s *AlibiExplainerSpec) GetContainer(metadata metav1.ObjectMeta, extensions *ComponentExtensionSpec, config *InferenceServicesConfig) *v1.Container {
	var args = []string{
		constants.ArgumentModelName, metadata.Name,
		constants.ArgumentPredictorHost, fmt.Sprintf("%s.%s", constants.DefaultPredictorServiceName(metadata.Name), metadata.Namespace),
		constants.ArgumentHttpPort, constants.InferenceServiceDefaultHttpPort,
	}
	if extensions.ContainerConcurrency != nil {
		args = append(args, constants.ArgumentWorkers, strconv.FormatInt(*extensions.ContainerConcurrency, 10))
	}
	if s.StorageURI != "" {
		args = append(args, "--storage_uri", constants.DefaultModelLocalMountPath)
	}

	args = append(args, string(s.Type))

	// Order explainer config map keys
	var keys []string
	for k, _ := range s.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--"+k)
		args = append(args, s.Config[k])
	}
	if s.Container.Image == "" {
		s.Image = config.Explainers.AlibiExplainer.ContainerImage + ":" + *s.RuntimeVersion
	}
	s.Name = constants.InferenceServiceContainerName
	s.Args = args
	return &s.Container
}

func (s *AlibiExplainerSpec) Default(config *InferenceServicesConfig) {
	s.Name = constants.InferenceServiceContainerName
	if s.RuntimeVersion == nil {
		s.RuntimeVersion = proto.String(config.Explainers.AlibiExplainer.DefaultImageVersion)
	}
	setResourceRequirementDefaults(&s.Resources)
}

// Validate the spec
func (s *AlibiExplainerSpec) Validate() error {
	return utils.FirstNonNilError([]error{
		validateStorageURI(s.GetStorageUri()),
	})
}

func (s *AlibiExplainerSpec) GetProtocol() constants.InferenceServiceProtocol {
	return constants.ProtocolV1
}

func (s *AlibiExplainerSpec) IsMMS(config *InferenceServicesConfig) bool {
	return false
}
