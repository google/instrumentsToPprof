# InstrumentsToPprof

This is a tool used to convert performance profiles from Xcode's Instruments tool on macOS to [pprof](http://github.com/google/pprof).

## Getting started

First clone the repo,

```
$ git clone https://github.com/google/instrumentsToPprof.git
```

The tool requires Go, which can be downloaded at the [Go homepage](https://golang.org/)

`instrumentsToPprof` can be installed to the `GOPATH` using
```
go install github.com/google/instrumentsToPprof
```

or run directly in the repo using
```
go run main.go
```

## Producing pprof from deep copy

The tool's input is the copied data from _Deep Copy_ inside Instruments. The _Deep Copy_
must be from the Time Profile tool in instruments, and the selection roots must be processes.

To get started, make a trace, either using `xctrace` or in the Instruments app.
```
$ xcrun -r xctrace record --template 'Time Profiler' --all-processes --time-limit 5s --output 'profile.trace'
```

Open the trace in the Instruments tool, and select the process that you want to have converted.
Multiple processes may be selected using `Cmd+Shift+C`. Then get the text data using _Deep Copy_
in the _Edit_ menu.

Paste the deep copy to a text file and run `instrumentsToPprof` which produces a file `profile.pb.gz`.
This file can analyzed using the [google/pprof](https://github.com/google/pprof) tool.

```
$ instrumentsToPprof deep_copy_paste.txt
```

Alternatively, one can produce the `profile.pb.gz` by piping the clipboard directly into `instrumentsToPprof`

```
$ pbpaste | instrumentsToPpof
```

## Producing a pprof from sample

`instrumentsToPprof` also supports output from the `sample` command on Mac.

To get a sample, run

```
$ sample <pid> -f <output-file>
```

and to produce a pprof from that sample, use the `--format` flag

```
$ instrumentsToPprof --format=sample <output-file>
```

# Disclaimer
This is not an officially supported Google product.
