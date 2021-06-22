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

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/instrumentsToPprof/internal"
	"github.com/google/instrumentsToPprof/internal/parsers"
)

const (
	help = `usage %[1]s [options] [deepcopy-file]
Converts a the deep copy output from Instrument's Time Profile tool to a pprof profile.

If deepcopy-file is empty, reads from stdin. To perform a conversion from the clipbaord, use
	$ pbpaste | %[1]s
Flags:
`
	formatHelp = `The format of the input. Use,
--format=sample for parsing sample files
--format=instruments for instruments deep-copy. This is the default.

Sample copying is a new feature and may have issues. File an issue on github in that case.
`
	pidTagHelp = `Annotated a process with pid with the given tag. Format is <pid>:<tag>.
For example, 'My Process Name [pid: 123] [Annotation]' with -pidTag=123:Annotation
`
)

const (
	kSample               string = "sample"
	kInstrumentsDeepCopy  string = "instruments"
	kInstrumentsCollapsed string = "collapsed"
)

type makeParserFn func(io.Reader) (parsers.Parser, error)

func main() {
	var outputFilename = flag.String("output", "profile.pb.gz", "Output file of the pprof profile.")
	var excludeProcessInStack = flag.Bool("exclude-process-from-stack",
		false, "Excludes processes from all stack traces.")
	var excludeThreadsInStack = flag.Bool("exclude-threads-from-stack",
		false, "Excludes threads from all stack traces.")
	var excludeIds = flag.Bool("exclude-ids", false, "Excludes ids from threads and processes")
	var format = flag.String("format", "instruments", formatHelp)
	var processAnnotations internal.ProcessAnnotationMap = make(map[uint64](string))
	flag.Var(&processAnnotations, "pidTag", pidTagHelp)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), help, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(-1)
	}
	inputFile := flag.Arg(0)

	var input io.Reader
	if inputFile == "-" || inputFile == "" {
		input = os.Stdin
	} else {
		file, err := os.Open(inputFile)
		if err != nil {
			log.Fatalf("Failed to open %s: %v", inputFile, err)
		}
		defer file.Close()
		input = file
	}

	var parserFn makeParserFn
	if *format == kSample {
		parserFn = parsers.MakeSampleParser
	} else if *format == kInstrumentsDeepCopy {
		parserFn = parsers.MakeDeepCopyParser
	} else if *format == kInstrumentsCollapsed {
		parserFn = parsers.MakeCollapsedParser
	} else {
		log.Fatalf("Invalid file format specified: %s", *format)
	}
	parser, err := parserFn(input)
	if err != nil {
		log.Fatal(err)
	}
	timeProfile, err := parser.ParseProfile()
	if err != nil {
		log.Fatalf("Failed to parse deep copy: %v", err)
	}
	pprof := internal.TimeProfileToPprof(timeProfile, *excludeProcessInStack,
		*excludeThreadsInStack, !*excludeIds, processAnnotations)
	if err = pprof.CheckValid(); err != nil {
		log.Fatalf("Invalid profile: %v\n", err)
	}
	out, err := os.Create(*outputFilename)
	if err != nil {
		log.Fatalf("output failed: %v", err)
	}
	defer out.Close()
	err = pprof.Write(out)
	if err != nil {
		log.Fatalf("failed to write: %v", err)
	}
}
