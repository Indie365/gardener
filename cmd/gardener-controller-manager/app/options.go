// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardener/cmd/utils"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	controllermanagerv1alpha1 "github.com/gardener/gardener/pkg/controllermanager/apis/config/v1alpha1"
	controllermanagervalidation "github.com/gardener/gardener/pkg/controllermanager/apis/config/validation"
	"github.com/gardener/gardener/pkg/features"
)

var configDecoder runtime.Decoder

func init() {
	configScheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		config.AddToScheme,
		controllermanagerv1alpha1.AddToScheme,
	)
	utilruntime.Must(schemeBuilder.AddToScheme(configScheme))
	configDecoder = serializer.NewCodecFactory(configScheme).UniversalDecoder()
}

type options struct {
	configFile string
	config     *config.ControllerManagerConfiguration
}

var _ utils.Options = &options{}

func (o *options) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.configFile, "config", o.configFile, "Path to configuration file.")
}

func (o *options) Complete() error {
	if len(o.configFile) == 0 {
		return errors.New("missing config file")
	}

	data, err := os.ReadFile(o.configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	o.config = &config.ControllerManagerConfiguration{}
	if err = runtime.DecodeInto(configDecoder, data, o.config); err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	// Set feature gates immediately after decoding the config.
	// Feature gates might influence the next steps, e.g., validating the config.
	return features.DefaultFeatureGate.SetFromMap(o.config.FeatureGates)
}

func (o *options) Validate() error {
	if errs := controllermanagervalidation.ValidateControllerManagerConfiguration(o.config); len(errs) > 0 {
		return errs.ToAggregate()
	}
	return nil
}

func (o *options) LogConfig() (string, string) {
	return o.config.LogLevel, o.config.LogFormat
}
