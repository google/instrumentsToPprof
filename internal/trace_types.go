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

package internal

import (
	"fmt"
	"strings"
)

// Frame represnets a frame in the stack.
type Frame struct {
	Parent       *Frame
	Children     []*Frame
	SelfWeightNs int64
	SymbolName   string
	Depth        int
}

func (f *Frame) String() string {
	space := strings.Repeat("  ", f.Depth)
	children_str := "{"
	for _, child := range f.Children {
		children_str += fmt.Sprintf("\n%s%s,", space, child)
	}
	children_str += "\n}"
	var parent_name = "nil"
	if f.Parent != nil {
		parent_name = f.Parent.SymbolName
	}
	return fmt.Sprintf("{SymbolName: %s Parent: %s Weight:%d Depth:%d Children:%s}",
		f.SymbolName, parent_name, f.SelfWeightNs, f.Depth, children_str)
}

// Thread represents the second level of the stack.
type Thread struct {
	Name   string
	Tid    uint64
	Frames []*Frame
}

func (t *Thread) String() string {
	return fmt.Sprintf("thread {name: %s tid: %d frames:\n%v\n]}", t.Name, t.Tid, t.Frames)
}

// Process are the top level of the stack.
type Process struct {
	Name    string
	Pid     uint64
	Threads []*Thread
}

func (p *Process) String() string {
	return fmt.Sprintf("process {name: %s pid: %d n_processes: %d}", p.Name, p.Pid, len(p.Threads))
}

// TimeProfile is a set of processes parsed from the deep copy.
type TimeProfile struct {
	Processes []*Process
}
