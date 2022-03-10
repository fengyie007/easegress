/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package filters

import (
	"fmt"

	"github.com/megaease/easegress/pkg/context"
	"github.com/megaease/easegress/pkg/supervisor"
	"github.com/megaease/easegress/pkg/util/yamltool"
	"github.com/megaease/easegress/pkg/v"
	"gopkg.in/yaml.v2"
)

type (
	// Filter is the interface of filters handling traffic of various protocols.
	Filter interface {
		// Name returns the name of the filter.
		Name() string

		// Kind returns the unique kind of the filter.
		Kind() string

		// DefaultSpec returns a spec for the filter, with default values. The
		// function should always return a new spec copy, because the caller
		// may modify the returned spec.
		DefaultSpec() Spec

		// Description returns the description of the filter.
		Description() string

		// Results returns all possible results, the normal result
		// (i.e. empty string) could not be in it.
		Results() []string

		// Init initializes the Filter.
		Init(spec Spec)

		// Inherit also initializes the Filter, the difference from Init is it
		// inherit something from the previousGeneration, but Inherit does NOT
		// handle the lifecycle of previousGeneration.
		Inherit(spec Spec, previousGeneration Filter)

		// Handle handles one HTTP request, all possible results
		// need be registered in Results.
		Handle(context.HTTPContext) (result string)

		// Status returns its runtime status.
		// It could return nil.
		Status() interface{}

		// Close closes itself.
		Close()
	}

	// Spec is the common interface of filter specs
	Spec interface {
		// Super returns supervisor
		Super() *supervisor.Supervisor

		// Name returns name.
		Name() string

		// Kind returns kind.
		Kind() string

		// Pipeline returns the name of the pipeline this filter belongs to.
		Pipeline() string

		// YAMLConfig returns the config in yaml format.
		YAMLConfig() string

		// baseSpec returns the pointer to the BaseSpec of the spec instance,
		// it is an internal function.
		baseSpec() *BaseSpec
	}

	// BaseSpec is the universal spec for all filters.
	BaseSpec struct {
		supervisor.MetaSpec `yaml:",inline"`
		super               *supervisor.Supervisor
		pipeline            string
		yamlConfig          string
	}
)

// NewSpec creates a filter spec and validates it.
func NewSpec(super *supervisor.Supervisor, pipeline string, rawSpec interface{}) (spec Spec, err error) {
	defer func() {
		if r := recover(); r != nil {
			spec = nil
			err = fmt.Errorf("%v", r)
		}
	}()

	yamlBuff, err := yaml.Marshal(rawSpec)
	if err != nil {
		return nil, err
	}

	// Meta part.
	meta := supervisor.MetaSpec{}
	if err = yaml.Unmarshal(yamlBuff, &meta); err != nil {
		return nil, err
	}
	if vr := v.Validate(&meta); !vr.Valid() {
		return nil, fmt.Errorf("%v", vr)
	}

	// Filter self part.
	root := GetRoot(meta.Kind)
	if root == nil {
		return nil, fmt.Errorf("kind %s not found", meta.Kind)
	}
	spec = root.DefaultSpec()
	if err = yaml.Unmarshal(yamlBuff, spec); err != nil {
		return nil, err
	}
	// TODO: Make the invalid part more accurate. e,g:
	// filters: jsonschemaErrs:
	// - 'policies.0: name is required'
	// to
	// filters: jsonschemaErrs:
	// - 'rateLimiter.policies.0: name is required'
	if vr := v.Validate(spec); !vr.Valid() {
		return nil, fmt.Errorf("%v", vr)
	}

	baseSpec := spec.baseSpec()
	baseSpec.super = super
	baseSpec.pipeline = pipeline
	baseSpec.yamlConfig = string(yamltool.Marshal(spec))
	return
}

// Super returns super
func (s *BaseSpec) Super() *supervisor.Supervisor {
	return s.super
}

// Name returns name.
func (s *BaseSpec) Name() string {
	return s.MetaSpec.Name
}

// Kind returns kind.
func (s *BaseSpec) Kind() string {
	return s.MetaSpec.Kind
}

// Pipeline returns the name of the pipeline this filter belongs to.
func (s *BaseSpec) Pipeline() string {
	return s.pipeline
}

// YAMLConfig returns the config in yaml format.
func (s *BaseSpec) YAMLConfig() string {
	return s.yamlConfig
}

func (s *BaseSpec) baseSpec() *BaseSpec {
	return s
}
