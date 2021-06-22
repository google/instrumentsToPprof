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

package collapsed

import (
  "strings"
	"testing"
)

func TestLineParsing(t *testing.T) {
  const collapsed = "Bar;Baz 2\n" +
  "Foo 2\n"

	r := strings.NewReader(collapsed)
	parser, err := MakeCollapsedParser(r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	parsed_profile, err := parser.ParseProfile()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

  // Only dummy process should be present.
  expected_process_count := 1
  if(len(parsed_profile.Processes) != expected_process_count){
		t.Errorf("Expected process count of %d", expected_process_count)
  }

  // Only dummy thread should be present.
  expected_thread_count := 1
  if(len(parsed_profile.Processes[0].Threads) != expected_thread_count){
		t.Errorf("Expected thread count of %d", expected_thread_count)
  }

  frames := parsed_profile.Processes[0].Threads[0].Frames
  expected_frame_count := 2
  if(len(frames) != expected_frame_count){
		t.Errorf("Expected frame count of %d, got, %d", expected_frame_count, len(frames))
  }

  // Keep track of how many of the expected frames we find.
  found_count := 0

  for _, frame := range frames{

    if frame.SymbolName == "Foo"{
      found_count+=1
      symbol_name := "Foo"

      if(frame.Depth != 0){
        t.Errorf("Wrong depth for %s. Got %d, expected %d", symbol_name, frame.Depth, 0)
      }

      if(len(frame.Children) != 0){
        t.Errorf("%s, should not have children frames, found %d", symbol_name, len(frame.Children))
      }
    }

    if frame.SymbolName == "Bar"{
      found_count+=1
      symbol_name := "Bar"

      if(frame.Depth != 0){
        t.Errorf("Wrong depth for %s. Got %d, expected %d", symbol_name, frame.Depth, 0)
      }

      if(len(frame.Children) != 1){
        t.Errorf("%s, should have 1 child frame, found %d", symbol_name, len(frame.Children))
      }

      child := frame.Children[0]
      child_symbol_name := child.SymbolName

      if(child_symbol_name != "Baz"){
        t.Errorf("Wrong symbol name for child of %s. Got %s, expected %s", symbol_name, child_symbol_name, "Baz")
      }

      if(child.Depth != 1){
        t.Errorf("Wrong depth for %s. Got %d, expected %d", child_symbol_name, frame.Depth, 0)
      }

      if(len(child.Children) != 0){
        t.Errorf("%s, should have 0 child frame, found %d", child_symbol_name, len(frame.Children))
      }

    }
  }

  if(found_count != 2){
    t.Errorf("Some of the expected frames were not found")
  }
}
