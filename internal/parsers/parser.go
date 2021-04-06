// Copyright 2020 Google LLC
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

package parsers

import (
	"io"

	"github.com/google/instrumentsToPprof/internal"
	"github.com/google/instrumentsToPprof/internal/parsers/instruments"
	"github.com/google/instrumentsToPprof/internal/parsers/sample"
)

type Parser interface {
	ParseProfile() (p *internal.TimeProfile, err error)
}

func MakeSampleParser(file io.Reader) (Parser, error) {
	return sample.MakeSampleParser(file)
}

func MakeDeepCopyParser(file io.Reader) (Parser, error) {
	d, err := instruments.MakeDeepCopyParser(file)
	if err != nil {
		return nil, err
	}
	return d, nil
}
