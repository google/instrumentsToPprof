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

package internal

import "testing"

func MakeDeepCopy() *TimeProfile {
	thread1 := &Thread{
		Name:   "thread1",
		Tid:    1,
		Frames: make([]*Frame, 1),
	}
	thread1.Frames[0] = &Frame{
		Parent:       nil,
		Children:     make([]*Frame, 1),
		SelfWeightNs: 0,
		SymbolName:   "first_frame",
		Depth:        2,
	}
	thread1.Frames[0].Children[0] = &Frame{
		Parent:       thread1.Frames[0],
		Children:     make([]*Frame, 0),
		SelfWeightNs: 1,
		SymbolName:   "sub_frame",
		Depth:        3,
	}
	process := &Process{
		Name:    "proc",
		Pid:     123,
		Threads: []*Thread{thread1},
	}
	return &TimeProfile{
		Processes: []*Process{process},
	}
}

var NoAnnotations ProcessAnnotationMap = make(map[uint64](string))

func TestIncludeProcessAndThreads(t *testing.T) {
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), false, false, true, NoAnnotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame -> thread1 -> proc
	if len(sample.Location) != 4 {
		t.Errorf("With both threads and processes, expected 4 frames. Was %v", sample.Location)
	}
	// Thread Frame #2
	if sample.Location[2].Line[0].Function.Name != "thread1 [tid: 0x1]" {
		t.Errorf("Expected thread at frame 2, was %v", sample.Location[2])
	}
	// Thread Frame #3
	if sample.Location[3].Line[0].Function.Name != "proc [pid: 123]" {
		t.Errorf("Expected process at frame 3, was %v", sample.Location[3])
	}
}

func TestIncludeProcessAndThreadsNoIds(t *testing.T) {
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), false, false, false, NoAnnotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame -> thread1 -> proc
	if len(sample.Location) != 4 {
		t.Errorf("With both threads and processes, expected 4 frames. Was %v", sample.Location)
	}
	// Thread Frame #2
	if sample.Location[2].Line[0].Function.Name != "thread1" {
		t.Errorf("Expected thread at frame 2, was %v", sample.Location[2])
	}
	// Thread Frame #3
	if sample.Location[3].Line[0].Function.Name != "proc" {
		t.Errorf("Expected process at frame 3, was %v", sample.Location[3])
	}
}

func TestExcludeThreads(t *testing.T) {
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), false, true, true, NoAnnotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame -> proc
	if len(sample.Location) != 3 {
		t.Errorf("With processes, expected 3 frames. Was %v", sample.Location)
	}
	// Process Frame #2
	if sample.Location[2].Line[0].Function.Name != "proc [pid: 123]" {
		t.Errorf("Expected process at frame 3, was %v", sample.Location[2])
	}
}

func TestExcludeProcesses(t *testing.T) {
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), true, false, true, NoAnnotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame -> thread1
	if len(sample.Location) != 3 {
		t.Errorf("With threads, expected 3 frames. Was %v", sample.Location)
	}
	// Thread Frame #2
	if sample.Location[2].Line[0].Function.Name != "thread1 [tid: 0x1]" {
		t.Errorf("Expected thread at frame 3, was %v", sample.Location[2])
	}
}

func TestExcludeProcessesAndThreads(t *testing.T) {
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), true, true, true, NoAnnotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame
	if len(sample.Location) != 2 {
		t.Errorf("With threads, expected 3 frames. Was %v", sample.Location)
	}
	// Check that we have the right parent frame.
	if sample.Location[1].Line[0].Function.Name != "first_frame" {
		t.Errorf("Expected frame at frame 1, was %v", sample.Location[1])
	}
}

func TestProcessAnnotations(t *testing.T) {
	annotations := make(map[uint64](string))
	annotations[123] = "MyAnnotation"
	annotations[1337] = "ExtraAnnotation"
	got := ConvertDeepCopyToProfile(MakeDeepCopy(), false, true, true, annotations)
	if len(got.Sample) != 1 {
		t.Errorf("Expected only 1 sample, got %v", got)
	}
	sample := got.Sample[0]
	// 4 frames: sub_frame -> first_frame
	if len(sample.Location) != 3 {
		t.Errorf("With processes, expected 3 frames. Was %v", sample.Location)
	}
	// Process Frame #2
	if sample.Location[2].Line[0].Function.Name != "proc [pid: 123] [MyAnnotation]" {
		t.Errorf("Expected process at frame 3, was %v", sample.Location[2])
	}
}
