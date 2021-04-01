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

package instruments

import (
	"strings"
	"testing"
)

func TestDeepCopyParsing(t *testing.T) {
	const deepCopy = "Weight\tSelf Weight\t\tSymbol Name\n" +
		"10.0 s  100%\t0 s\t \tMain Process (123)\n" +
		"5.0 s  50%\t0 s\t \t Thread 1  0x1ee7\n" +
		"5.0 s  50%\t0 s\t \t  foo\n" +
		"2.0 s  20%\t2.0 s\t \t   bar1\n" +
		"3.0 s  30%\t1.0 s\t \t   bar2\n" +
		"2.0 s  20%\t2.0 s\t \t    baz\n" +
		"5.0 s  50%\t0 s\t \t Thread 2  0x7ee1\n" +
		"5.0 s  50%\t5.0 s\t \t  spin\n" +
		"\n"

	r := strings.NewReader(deepCopy)
	got, err := ParseDeepCopy(r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Test got
	if len(got.Processes) != 1 {
		t.Errorf("Only got %d processes %v", len(got.Processes), got.Processes)
	}
	proc := got.Processes[0]
	if len(proc.Threads) != 2 {
		t.Errorf("Didn't get 2 processes %v", proc.Threads)
	}
	if proc.Pid != 123 || proc.Name != "Main Process" {
		t.Errorf("Process was wrong. Got %v, expected pid=%d name=%s", proc, 123, "Main Process")
	}
	th := proc.Threads[0]
	if th.Tid != 0x1ee7 || th.Name != "Thread 1" {
		t.Errorf("Thread was wrong. Got %v, expected tid=%d name=%s", th, 0x1ee7, "Thread 1")
	}
	foo := th.Frames[0]
	bar2 := foo.Children[1]
	if bar2.Parent != foo {
		t.Errorf("Parents not setup %v => %v", bar2, foo)
	}
	baz := bar2.Children[0]
	if baz.SelfWeightNs != 2_000_000_000 {
		t.Errorf("baz should have self weight %d was %d", 2_000_000_000, baz.SelfWeightNs)
	}
}

func TestInvalidThreadAndProcessNames(t *testing.T) {
	const deepCopy = "Weight\tSelf Weight\t\tSymbol Name\n" +
		"10.0 s  100%\t0 s\t \tMain Process 123\n" +
		"5.0 s  50%\t0 s\t \t Thread 1 0x1ee7\n" +
		"5.0 s  50%\t0 s\t \t  foo\n" +
		"2.0 s  20%\t2.0 s\t \t   bar1\n" +
		"3.0 s  30%\t1.0 s\t \t   bar2\n" +
		"2.0 s  20%\t2.0 s\t \t    baz\n" +
		"5.0 s  50%\t0 s\t \t Thread 2  0x7ee1\n" +
		"5.0 s  50%\t5.0 s\t \t  spin\n" +
		"\n"

	r := strings.NewReader(deepCopy)
	got, err := ParseDeepCopy(r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// Parsing failed, fallback to full name.
	if got.Processes[0].Name != "Main Process 123" {
		t.Errorf("Expected process name %s was %s", "Main Process 123", got.Processes[0].Name)
	}
	if got.Processes[0].Threads[0].Name != "Thread 1 0x1ee7" {
		t.Errorf("Expected thread name %s was %s", "Thread 1 0x1ee7", got.Processes[0].Threads[0].Name)
	}
}
