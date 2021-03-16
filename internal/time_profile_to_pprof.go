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

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
)

// ProcessAnnotationMap used for renaming the process based on pid.
type ProcessAnnotationMap map[uint64](string)

func (m *ProcessAnnotationMap) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *ProcessAnnotationMap) Set(value string) error {
	// Format of string is <pid>:<annotation>
	sp := strings.SplitN(value, ":", 2)
	pid, err := strconv.ParseUint(sp[0], 10, 64)
	if err != nil {
		return err
	}
	annotation := sp[1]
	old, ok := (*m)[pid]
	if ok {
		return fmt.Errorf("Duplicate annotation found on pid %d: %s", pid, old)
	}
	(*m)[pid] = annotation
	return nil
}

type location struct {
	pid        uint64
	tid        uint64
	methodName string
}

type deepCopyToPprofConverter struct {
	deepCopy *TimeProfile
	// Settings
	excludeProcessesFromStack  bool
	excludeThreadsFromStack    bool
	includeThreadAndProcessIds bool
	annotations                ProcessAnnotationMap
	consumedAnnotations        ProcessAnnotationMap

	// functions by name
	functions      map[string]*profile.Function
	nextFunctionID uint64
	locations      map[location]*profile.Location
	nextLocationID uint64

	samples []*profile.Sample
}

func newPprofConverter(
	deepCopy *TimeProfile,
	excludeProcessesFromStack bool,
	excludeThreadsFromStack bool,
	includeThreadAndProcessIds bool,
	annotations ProcessAnnotationMap) *deepCopyToPprofConverter {
	return &deepCopyToPprofConverter{
		deepCopy:                   deepCopy,
		excludeProcessesFromStack:  excludeProcessesFromStack,
		excludeThreadsFromStack:    excludeThreadsFromStack,
		includeThreadAndProcessIds: includeThreadAndProcessIds,
		annotations:                annotations,
		consumedAnnotations:        make(map[uint64](string)),
		functions:                  make(map[string]*profile.Function),
		nextFunctionID:             1,
		locations:                  make(map[location]*profile.Location),
		nextLocationID:             1,
		samples:                    make([]*profile.Sample, 0),
	}
}

func (toPprof *deepCopyToPprofConverter) getFunction(name string) *profile.Function {
	f, ok := toPprof.functions[name]
	if !ok {
		f = &profile.Function{
			ID:         toPprof.nextFunctionID,
			Name:       name,
			SystemName: name,
		}
		toPprof.functions[name] = f
		toPprof.nextFunctionID++
		return f
	}
	return f
}

func (toPprof *deepCopyToPprofConverter) getLocation(symbolName string, proc *Process, th *Thread) *profile.Location {
	id := location{methodName: symbolName, pid: proc.Pid, tid: th.Tid}
	loc, ok := toPprof.locations[id]
	if !ok {
		loc = &profile.Location{
			ID:   toPprof.nextLocationID,
			Line: []profile.Line{{Function: toPprof.getFunction(symbolName)}},
		}
		toPprof.locations[id] = loc
		toPprof.nextLocationID++
		return loc
	}
	return loc
}

func (toPprof *deepCopyToPprofConverter) getThreadLocation(proc *Process, th *Thread) *profile.Location {
	var name string
	if toPprof.includeThreadAndProcessIds {
		name = fmt.Sprintf("%s [tid: 0x%x]", th.Name, th.Tid)
	} else {
		name = th.Name
	}
	id := location{methodName: name, pid: proc.Pid, tid: th.Tid}
	loc, ok := toPprof.locations[id]
	if !ok {
		loc = &profile.Location{
			ID:   toPprof.nextLocationID,
			Line: []profile.Line{{Function: toPprof.getFunction(name)}},
		}
		toPprof.locations[id] = loc
		toPprof.nextLocationID++
		return loc
	}
	return loc
}

func (toPprof *deepCopyToPprofConverter) getProcessLocation(proc *Process) *profile.Location {
	var name string
	if toPprof.includeThreadAndProcessIds {
		name = fmt.Sprintf("%s [pid: %d]", proc.Name, proc.Pid)
	} else {
		name = proc.Name
	}
	// Skip unparsable pids.
	if proc.Pid != 0 {
		annotation, ok := toPprof.annotations[proc.Pid]
		if ok {
			toPprof.consumedAnnotations[proc.Pid] = annotation
			name = fmt.Sprintf("%s [%s]", name, annotation)
		}
	}
	id := location{methodName: proc.Name, pid: proc.Pid, tid: 0}
	loc, ok := toPprof.locations[id]
	if !ok {
		loc = &profile.Location{
			ID:   toPprof.nextLocationID,
			Line: []profile.Line{{Function: toPprof.getFunction(name)}},
		}
		toPprof.locations[id] = loc
		toPprof.nextLocationID++
		return loc
	}
	return loc
}

func (toPprof *deepCopyToPprofConverter) convertSample(sample *Frame, th *Thread, proc *Process) *profile.Sample {
	stackTrace := make([]*profile.Location, 0)
	currentFrame := sample
	for {
		if currentFrame == nil {
			break
		}
		stackTrace = append(stackTrace, toPprof.getLocation(currentFrame.SymbolName, proc, th))
		currentFrame = currentFrame.Parent
	}
	if !toPprof.excludeThreadsFromStack {
		stackTrace = append(stackTrace, toPprof.getThreadLocation(proc, th))
	}
	if !toPprof.excludeProcessesFromStack {
		stackTrace = append(stackTrace, toPprof.getProcessLocation(proc))
	}
	return &profile.Sample{
		Location: stackTrace,
		Value:    []int64{sample.SelfWeightNs},
		Label: map[string][]string{
			"pid":          {strconv.FormatUint(proc.Pid, 10)},
			"tid":          {strconv.FormatUint(th.Tid, 10)},
			"process_name": {proc.Name},
			"thread_name":  {th.Name},
		},
	}
}

func (toPprof *deepCopyToPprofConverter) findSamplesInFrame(proc *Process, th *Thread, currentFrame *Frame) {
	if currentFrame.SelfWeightNs != 0 {
		toPprof.samples = append(toPprof.samples, toPprof.convertSample(currentFrame, th, proc))
	}
	for _, f := range currentFrame.Children {
		toPprof.findSamplesInFrame(proc, th, f)
	}
}

func (toPprof *deepCopyToPprofConverter) findSamples(proc *Process, th *Thread) {
	if len(th.Frames) == 0 {
		return
	}
	for _, currentFrame := range th.Frames {
		toPprof.findSamplesInFrame(proc, th, currentFrame)
	}
}

func (toPprof *deepCopyToPprofConverter) convertToPprof() *profile.Profile {
	for _, proc := range toPprof.deepCopy.Processes {
		for _, th := range proc.Threads {
			toPprof.findSamples(proc, th)
		}
	}

	locations := make([]*profile.Location, len(toPprof.locations))
	i := 0
	for _, loc := range toPprof.locations {
		locations[i] = loc
		i++
	}
	functions := make([]*profile.Function, len(toPprof.functions))
	i = 0
	for _, fn := range toPprof.functions {
		functions[i] = fn
		i++
	}

	if len(toPprof.consumedAnnotations) < len(toPprof.annotations) {
		warning := "Not all annotations were used. The following pids could not be found:"
		for pid, annotation := range toPprof.annotations {
			if _, ok := toPprof.consumedAnnotations[pid]; !ok {
				warning += fmt.Sprintf("\n  %d: %s", pid, annotation)
			}
		}
		fmt.Printf("WARNING: %s\n", warning)
	}
	return &profile.Profile{
		SampleType: []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
		Sample:     toPprof.samples,
		Location:   locations,
		Function:   functions,
	}
}

// TimeProfileToPprof converts a TimeProfile to a pprof Profile.
func TimeProfileToPprof(deepCopy *TimeProfile,
	excludeProcessesFromStack bool,
	excludeThreadsFromStack bool,
	includeThreadAndProcessIds bool,
	annotations ProcessAnnotationMap) *profile.Profile {
	converter := newPprofConverter(deepCopy, excludeProcessesFromStack, excludeThreadsFromStack, includeThreadAndProcessIds, annotations)
	if excludeProcessesFromStack && len(annotations) > 0 {
		fmt.Println("WARNING: Combined annotations with excluding process from the stack. Annotations will be ignored.")
	}
	return converter.convertToPprof()
}
