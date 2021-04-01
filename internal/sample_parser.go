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
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var (
	functionRe = regexp.MustCompile(`([+\s!:|]*)(\d+)\s+(.*)$`)
)

func parseCallLine(line string) (f *Frame, err error) {
	matches := functionRe.FindStringSubmatch(line)
	if matches == nil || len(matches) != 4 {
		return nil, fmt.Errorf("Failed to parse function line: %s", line)
	}
	hits, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Error parsing function line %s: %v", line, err)
	}

	return &Frame{
		SymbolName:   matches[3],
		SelfWeightNs: hits,
		// 2 spaces per depth.
		Depth: len(matches[1]) / 2,
	}, nil
}

func fixSelfWeight(frame *Frame) error {
	for _, child := range frame.Children {
		frame.SelfWeightNs -= child.SelfWeightNs
		if frame.SelfWeightNs < 0 {
			return fmt.Errorf(
				"Fatal error parsing sample file. Frame %s had negative weight. The file is either corrupt or this is a bug.",
				frame.SymbolName)
		}
		fixSelfWeight(child)
	}
	return nil
}

var (
	pidRe = regexp.MustCompile(`(.*)\s\[(\d+)\]`)
)

func parseProcess(line string) (p *Process, err error) {
	// Parse process line, which looks like,
	// Process:         Google Chrome Helper (Renderer) [56690]
	invalid_line := fmt.Errorf("Not valid process line %s", line)
	if !strings.HasPrefix(line, "Process") {
		return nil, invalid_line
	}
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return nil, invalid_line
	}
	pid_part := strings.TrimSpace(parts[1])
	matches := pidRe.FindStringSubmatch(pid_part)
	if matches == nil || len(matches) != 3 {
		return nil, fmt.Errorf("Error parsing process and pid from %s: %v", pid_part, matches)
	}
	pid, err := strconv.ParseUint(matches[2], 10, 64)
	return &Process{
		Pid:  pid,
		Name: matches[1],
	}, nil
}

func parseSampleRate(line string) int64 {
	parts := strings.Split(line, " ")
	n := len(parts)
	unit := parts[n-1]
	period := parts[n-2]
	// TODO(eshrubs): Implement frequency parsing.
	if period != "1" && unit != "millisecond" {
		log.Printf(
			"WARNING: Period parsing is not yet supported. Defaulting to 1ms period but period of %s %s was detected",
			period, unit)
	}
	return 1_000_000
}

func ParseSample(file io.Reader) (p *TimeProfile, err error) {
	p = &TimeProfile{}

	buf := bufio.NewReader(file)

	// Default sample rate of 1ms == 1,000,000 ns
	var sampleRate int64 = 1_000_000
	// TODO(eshr): Parse sample rate
	// Parse header
	for {
		line, err := buf.ReadString('\n')
		if line == "" && err != nil {
			// Break once end of file.
			if err == io.EOF {
				return nil, errors.New("Could not create trace, could not find the 'Call Graph' line.")
			}
			return nil, err
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Analysis of sampling") {
			sampleRate = parseSampleRate(line)
		}
		if strings.HasPrefix(line, "Report Version") {
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("Could not parse report version line: %s", line)
			}
			reportVersion, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("Error parsing report version: %v", err)
			}
			if reportVersion != 7 {
				return nil, fmt.Errorf("Report Version was %d, only report version 7 is supported", reportVersion)
			}
		}
		if strings.HasPrefix(line, "Process") {
			if len(p.Processes) > 0 {
				return nil, errors.New("More than one process line present. Currupt sample file")
			}
			process, err := parseProcess(line)
			if err != nil {
				return nil, err
			}
			p.Processes = append(p.Processes, process)
		}
		if strings.HasPrefix(line, "Call graph") {
			break
		}
	}
	process := p.Processes[0]
	var currentThread *Thread = nil
	var lastFrame *Frame = nil
	for {
		line, err := buf.ReadString('\n')
		if line == "" && err != nil {
			// Break once end of file.
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line)

		// Call stack is over
		if line == "" {
			break
		}

		// Parse a function.
		currentFrame, err := parseCallLine(line)
		if err != nil {
			return nil, err
		}
		if currentFrame.Depth == 0 {
			// New thread!
			currentThread = &Thread{
				Name: currentFrame.SymbolName,
			}
			process.Threads = append(process.Threads, currentThread)
		} else if currentFrame.Depth == 1 {
			// First frame in thread
			currentThread.Frames = append(currentThread.Frames, currentFrame)
		} else if currentFrame.Depth > lastFrame.Depth {
			// Child frame
			if currentFrame.Depth-lastFrame.Depth != 1 {
				return nil, fmt.Errorf("Skipped frame depth from frame %s to %s",
					lastFrame.SymbolName, currentFrame.SymbolName)
			}
			lastFrame.Children = append(lastFrame.Children, currentFrame)
			currentFrame.Parent = lastFrame
		} else {
			// Find parent
			var parent *Frame = lastFrame.Parent
			for {
				if parent.Depth == currentFrame.Depth-1 {
					parent.Children = append(parent.Children, currentFrame)
					currentFrame.Parent = parent
					break
				}
				parent = parent.Parent
			}
		}
		currentFrame.SelfWeightNs *= sampleRate
		lastFrame = currentFrame
	}

	// Fix weights
	for _, thread := range process.Threads {
		for _, frame := range thread.Frames {
			err := fixSelfWeight(frame)
			if err != nil {
				return nil, err
			}
		}
	}

	return p, nil
}
