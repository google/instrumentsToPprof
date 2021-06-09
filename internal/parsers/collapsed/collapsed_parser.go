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

package collapsed

import (
	"bufio"
  //"fmt"
  "io"
  "strconv"
  "strings"

  "github.com/google/instrumentsToPprof/internal"
)

func MakeCollapsedParser(file io.Reader) (d CollapsedParser, err error) {
	d = CollapsedParser{
		lines: []string{},
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		d.lines = append(d.lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return d, err
	}
	return d, nil
}

type CollapsedParser struct {
	lines []string
}

func (d CollapsedParser) ParseProfile() (p *internal.TimeProfile, err error) {
	// TODO: Implement parsing in the struct.
	p = &internal.TimeProfile{}
  
  // Collapsed format merges all processes so just create a dummy one.
  var process = &internal.Process{
		Pid:  0,
		Name: "",
	}
  p.Processes = append(p.Processes, process)

  // Collapsed format merges all threads so just create a dummy one.
  var currentThread = &internal.Thread{
    Name: "",
  }

  process.Threads = append(process.Threads, currentThread)

	for _, line := range d.lines {
		currentFrame, err := parseCallLine(line)
    if err == nil {
      currentThread.Frames = append(currentThread.Frames, currentFrame)
    }
  }

	return p, nil
}

func parseCallLine(line string) (f *internal.Frame, err error) {

  sep := strings.LastIndex(line, " ") 

  frequence, err := strconv.ParseInt(line[sep+1:len(line)], 10, 64)
  funs := strings.Split(line[0:sep], ";")

	var frame = &internal.Frame{
		SymbolName:  funs[0],
		SelfWeightNs: 0,
		Depth: 0,
	}

  var last_frame *internal.Frame = frame;
	for index, fun := range funs{
    if index == 0 {
      continue
    }

    var current_frame = &internal.Frame{
      SymbolName:  fun,
      SelfWeightNs: 0,
      Depth: index,
    }

    current_frame.Parent = last_frame
    last_frame.Children = append(last_frame.Children, current_frame)
    last_frame = current_frame
  }

  last_frame.SelfWeightNs = frequence

  return frame, nil
}
