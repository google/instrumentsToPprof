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
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/instrumentsToPprof/internal"
)

func MakeDeepCopyParser(file io.Reader) (d DeepCopyParser, err error) {
	d = DeepCopyParser{
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

type DeepCopyParser struct {
	lines []string
}

func (d DeepCopyParser) ParseProfile() (p *internal.TimeProfile, err error) {
	// TODO: Implement parsing in the struct.
	p = &internal.TimeProfile{}

	// First line must match header
	// Now parse away since first line was good.
	var lastFrame *internal.Frame = nil
	var currentProcess *internal.Process = nil
	var currentThread *internal.Thread = nil
	for _, line := range d.lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Process end. Start again with new process.
			currentProcess = nil
			currentThread = nil
			lastFrame = nil
			continue
		}
		// Try to fetch process
		if currentProcess == nil {
			// Header line
			if line == "Weight\tSelf Weight\t\tSymbol Name" || line == "Weight\tSelf Weight\tSymbol Names" {
				continue
			}
			f, err := parseLine(line)
			if err != nil {
				return nil, fmt.Errorf("Error parsing process frame: %v", err)
			}
			currentProcess, err = newProcessFromFrame(f)
			if err != nil {
				return nil, err
			}
			p.Processes = append(p.Processes, currentProcess)
		} else if currentThread == nil {
			f, err := parseLine(line)
			if err != nil {
				return nil, fmt.Errorf("Error parsing thread frame: %v", err)
			}
			currentThread, err = newThreadFromFrame(f)
			if err != nil {
				return nil, err
			}
			currentProcess.Threads = append(currentProcess.Threads, currentThread)
		} else {
			// Parse frame
			currentFrame, err := parseLine(line)
			if err != nil {
				return nil, err
			}
			if currentFrame.Depth == 0 {
				return nil, fmt.Errorf("Unexpected new process, should have occurred after header line %s", line)
			}
			if currentFrame.Depth == 1 {
				// New thread
				currentThread, err = newThreadFromFrame(currentFrame)
				if err != nil {
					return nil, fmt.Errorf("Error parsing thread frame: %v", err)
				}
				currentProcess.Threads = append(currentProcess.Threads, currentThread)
				lastFrame = nil
				continue
			}
			if lastFrame == nil {
				// First frame in thread.
				if currentFrame.Depth != 2 {
					return nil, fmt.Errorf("First frame in thread should have depth 2, was %d: %s", currentFrame.Depth, line)
				}
				currentThread.Frames = append(currentThread.Frames, currentFrame)
				lastFrame = currentFrame
				continue
			}
			if currentFrame.Depth == 2 {
				// New thread frame, this will be a parent frame.
				currentThread.Frames = append(currentThread.Frames, currentFrame)
				lastFrame = currentFrame
				continue
			}
			if currentFrame.Depth > lastFrame.Depth {
				if currentFrame.Depth-lastFrame.Depth != 1 {
					return nil, fmt.Errorf("Skip children somehow?: %s", line)
				}
				lastFrame.Children = append(lastFrame.Children, currentFrame)
				currentFrame.Parent = lastFrame
			} else {
				// Find parent
				var parent *internal.Frame = lastFrame.Parent
				for {
					if parent.Depth == currentFrame.Depth-1 {
						parent.Children = append(parent.Children, currentFrame)
						currentFrame.Parent = parent
						break
					}
					parent = parent.Parent
				}
			}
			lastFrame = currentFrame
		}
	}
	return p, nil
}

func newThreadFromFrame(f *internal.Frame) (*internal.Thread, error) {
	if f.Depth != 1 {
		return nil, fmt.Errorf("Thread must have depth 1, was %d: %v", f.Depth, f)
	}
	// Thread name is in format "<thread name>  0x<tid>"
	threadRe := regexp.MustCompile(`(.*)\s\s0x([0-9a-f]+)$`)
	matches := threadRe.FindStringSubmatch(f.SymbolName)
	if len(matches) != 3 {
		fmt.Printf("WARNING: Error parsing thread '%s'. Skipping thread name parsing.\n", f.SymbolName)
		return &internal.Thread{
			Name:   f.SymbolName,
			Tid:    0,
			Frames: make([]*internal.Frame, 0),
		}, nil
	}
	tid, err := strconv.ParseUint(matches[2], 16, 64)
	if err != nil {
		fmt.Printf("WARNING: Error parsing tid '%s'. Skipping thread id parsing. %v\n", matches[2], err)
		tid = 0
	}
	return &internal.Thread{
		Name:   matches[1],
		Tid:    tid,
		Frames: make([]*internal.Frame, 0),
	}, nil
}

func newProcessFromFrame(f *internal.Frame) (*internal.Process, error) {
	if f.Depth != 0 {
		return nil, fmt.Errorf("Process must have depth 0, was %d: %v", f.Depth, f)
	}
	// Process name is in format "<process name> (<pid>)"
	processRe := regexp.MustCompile(`(.*)\s\((\d+)\)$`)
	matches := processRe.FindStringSubmatch(f.SymbolName)
	if len(matches) != 3 {
		fmt.Printf("WARNING: Error parsing process '%s'. Skipping process name parsing.\n", f.SymbolName)
		return &internal.Process{
			Name:    f.SymbolName,
			Pid:     0,
			Threads: make([]*internal.Thread, 0),
		}, nil
	}
	pid, err := strconv.ParseUint(matches[2], 10, 64)
	if err != nil {
		fmt.Printf("WARNING: Error parsing pid '%s'. Skipping process id parsing. %v\n", matches[2], err)
		pid = 0
	}
	return &internal.Process{
		Name:    matches[1],
		Pid:     pid,
		Threads: make([]*internal.Thread, 0),
	}, nil
}

func parseSelfWeight(selfWeightText string) (int64, error) {
	// String is in the format "2.00 ms" where valid units
	// that I know about are "s", "ms", "µs", and "ns".
	// returns nanoseconds.

	fields := strings.Split(selfWeightText, " ")
	if len(fields) != 2 {
		return 0, fmt.Errorf("Self weight not parsable: was not 2 fields in \"%s\"", selfWeightText)
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("Could not parse self weight %s: %v", selfWeightText, err)
	}
	switch fields[1] {
	case "s":
		value *= 1_000_000_000
	case "ms":
		value *= 1_000_000
	case "µs":
		value *= 1_000
	case "ns":
		value *= 1
	default:
		return 0, fmt.Errorf("Could not interpret time unit '%s' in %s", selfWeightText, fields[1])
	}

	return int64(value), nil
}

func parseLine(line string) (*internal.Frame, error) {
	// Each line is tab separated into 3 or 4 fields
	// 1. Total weight "254.00 ms   22.5%"
	// 2. Self weight "2.00ms"
	// 3. Optionally, a space
	// 4. Depth (leading spaces) + Symbol name "    foo"
	fields := strings.Split(line, "\t")
	if len(fields) != 3 && len(fields) != 4 {
		return nil, fmt.Errorf(
			"Could not parse line \"%s\", only found %d tab-separated fields",
			line, len(fields))
	}
	weight, err := parseSelfWeight(fields[1])
	if err != nil {
		return nil, err
	}
	lastField := fields[len(fields) - 1]
	name := strings.TrimLeft(lastField, " ")
	depth := len(lastField) - len(name)
	if len(fields) == 3 {
		depth -= 1
	}
	return &internal.Frame{
		Parent:       nil,
		Children:     make([]*internal.Frame, 0),
		SelfWeightNs: weight,
		SymbolName:   name,
		Depth:        depth,
	}, nil
}
