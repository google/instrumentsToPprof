// Copyright 2021 Google LLC
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

package sample

import (
	"strings"
	"testing"

	"github.com/google/instrumentsToPprof/internal"
)

const (
	validDeepCopy = `Analysis of sampling Process Name (pid 56690) every 1 millisecond
Process:         ProcessName [56690]
Path:            /Applications/Process.app/Contents/Frameworks/
Load Address:    0x10b6ed000
Identifier:      ProcessName
Version:         ???
Code Type:       X86-64
Platform:        macOS

Date/Time:       2021-03-15 15:41:58.406 +0100
Launch Time:     2021-03-15 15:30:30.917 +0100
OS Version:      macOS 11.2.2 (20D80)
Report Version:  7
Analysis Tool:   /usr/bin/sample

Physical footprint:         530.5M
Physical footprint (peak):  538.4M
----

Call graph:
    4 Thread1 DispatchQueue1: com.apple.main-thread  (serial)
    + 4 start
    +   4 eatLunch
    +   : 3 makeSandwhich
    +   : ! 1 getBread(BreadType)
    +   : !	1 getCheese(CheeseType)
    +   : 1 eatFood(Food const&)
    1 Thread2 DispatchQueue1: com.apple.main-thread  (serial)
    + 1 listenToMusic()
		`
)

func TestSampleParsing(t *testing.T) {
	r := strings.NewReader(validDeepCopy)
	parser, err := MakeSampleParser(r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	timeProfile, err := parser.ParseProfile()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	expected := &internal.TimeProfile{
		Processes: []*internal.Process{
			{
				Name: "ProcessName",
				Pid:  56690,
				Threads: []*internal.Thread{
					{
						Name: "Thread1 DispatchQueue1: com.apple.main-thread  (serial)",
						Tid:  0,
						Frames: []*internal.Frame{
							{
								SymbolName:   "start",
								Depth:        1,
								SelfWeightNs: 0,
								Children: []*internal.Frame{
									{
										SymbolName:   "eatLunch",
										Depth:        2,
										SelfWeightNs: 0,
										Children: []*internal.Frame{
											{
												SymbolName:   "makeSandwhich",
												Depth:        3,
												SelfWeightNs: 1_000_000,
												Children: []*internal.Frame{
													{
														SymbolName:   "getBread(BreadType)",
														Depth:        4,
														SelfWeightNs: 1_000_000,
														Children:     []*internal.Frame{},
													}, {
														SymbolName:   "getCheese(CheeseType)",
														Depth:        4,
														SelfWeightNs: 1_000_000,
														Children:     []*internal.Frame{},
													},
												},
											},
											{
												SymbolName:   "eatFood(Food const&)",
												Depth:        3,
												SelfWeightNs: 1_000_000,
												Children:     []*internal.Frame{},
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "Thread2 DispatchQueue1: com.apple.main-thread  (serial)",
						Tid:  0,
						Frames: []*internal.Frame{
							{
								SymbolName:   "listenToMusic()",
								Depth:        1,
								SelfWeightNs: 1_000_000,
								Children:     []*internal.Frame{},
							},
						},
					},
				},
			},
		},
	}

	internal.TimeProfileEquals(t, timeProfile, expected)
}
