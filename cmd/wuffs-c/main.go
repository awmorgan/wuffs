// Copyright 2017 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

// ----------------

// wuffs-c handles the C language specific parts of the wuffs tool.
package main

import (
	"fmt"
	"os"

	"github.com/google/wuffs/internal/cgen"
)

func main() {
	if err := main1(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

const usage = `wuffs-c is the C language backend for Wuffs.

Usage:
  wuffs-c <sub-command> [flags] [files...]

Available Sub-commands:
  gen        Transpiles Wuffs source files into C code.
  test       Transpiles and runs C tests for Wuffs files.
  bench      Transpiles and runs benchmarks.
  genlib     Generates static C library outputs.
  genrelease Builds monolithic release headers.

Examples:
  wuffs-c gen -package_name=main main.wuffs > app.c

Packages named main automatically receive the standalone C main wrapper.
`

func main1() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("%s\nerror: no sub-command given", usage)
	}
	args := os.Args[2:]
	switch os.Args[1] {
	case "help", "-h", "-help", "--help":
		fmt.Print(usage)
		return nil
	case "bench":
		return doBench(args)
	case "gen":
		return cgen.Do(args)
	case "genlib":
		return doGenlib(args)
	case "genrelease":
		return doGenrelease(args)
	case "test":
		return doTest(args)
	}
	return fmt.Errorf("%s\nerror: bad sub-command %q", usage, os.Args[1])
}
