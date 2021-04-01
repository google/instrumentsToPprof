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

// ParseDeepCopy parses the deep copy from the input.
package internal

import (
	"testing"
)

func frameEquals(t *testing.T, a *Frame, b *Frame) {
	t.Helper()
	if (a == nil && b == nil) {
		return;
	}
	if (a == nil || b == nil) {
		t.Fatalf("One frame was nil %v != %v", a, b)
	}
	if (a.SymbolName != b.SymbolName) {
		t.Errorf("SymbolName %s != %s", a.SymbolName, b.SymbolName)
	}
	if (a.Depth != b.Depth) {
		t.Errorf("%s frame depth %d != %d", a.SymbolName, a.Depth, b.Depth)
	}
	if (a.SelfWeightNs != b.SelfWeightNs) {
		t.Errorf("%s self weight %d != %d", a.SymbolName, a.SelfWeightNs, b.SelfWeightNs)
	}
	if (len(a.Children) != len(b.Children)) {
		t.Fatalf("%s have different children lengths %v != %v", a.SymbolName, a.Children, b.Children)
	}
	for i, aChild := range a.Children {
		bChild := b.Children[i]
		frameEquals(t, aChild, bChild)
	}
}

func threadEquals(t *testing.T, a *Thread, b *Thread) {
	t.Helper()
	if (a.Name != b.Name) {
		t.Errorf("Threads have different names %s != %s", a.Name, b.Name)
	}
	if (a.Tid != b.Tid) {
		t.Errorf("Thread %s have different tids %d != %d", a.Name, a.Tid, b.Tid)
	}
	if len(a.Frames) != len(b.Frames) {
		t.Fatalf("Thread %s have different number of frames %v != %v", a.Name, a.Frames, b.Frames)
	}
	for i, aChild := range a.Frames {
		bChild := b.Frames[i]
		frameEquals(t, aChild, bChild)
	}
}

func processEquals(t *testing.T, a *Process, b *Process) {
	t.Helper()
	if (a.Name != b.Name) {
		t.Errorf("Processes have diferent Names %s != %s", a.Name, b.Name)
	}
	if (a.Pid != b.Pid) {
		t.Errorf("Process has different pids %d != %d", a.Pid, b.Pid)
	}
	if len(a.Threads) != len(b.Threads) {
		t.Fatalf("Processes have different number of threads %d != %d",
		len(a.Threads), len(b.Threads))
	}
	for i, aThread := range a.Threads {
		bThread := b.Threads[i]
		threadEquals(t, aThread, bThread)
	}
}

func timeProfileEquals(t *testing.T, a *TimeProfile, b *TimeProfile) {
	t.Helper()
	if (len(a.Processes) != len(b.Processes)) {
		t.Fatalf("Time profiles had different number of processes %d != %d",
		len(a.Processes), len(b.Processes))
	}
	for i, aProcess := range a.Processes {
		bProcess := b.Processes[i]
		processEquals(t, aProcess, bProcess)
	}
}
