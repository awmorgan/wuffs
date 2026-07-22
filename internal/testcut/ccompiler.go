// Copyright 2026 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package testcut

import (
	"fmt"
	"os"
	"os/exec"
)

type CCompiler struct {
	Path   string
	Flavor string // "clang", "gcc", "cl"
}

// FindCCompiler detects an installed C compiler for building general-purpose Wuffs applications.
// Checks CC environment variable first, then searches PATH for clang, gcc, or cl.
func FindCCompiler() (*CCompiler, error) {
	if cc := os.Getenv("CC"); cc != "" {
		if path, err := exec.LookPath(cc); err == nil {
			return &CCompiler{Path: path, Flavor: "env"}, nil
		}
	}

	if path, err := exec.LookPath("clang"); err == nil {
		return &CCompiler{Path: path, Flavor: "clang"}, nil
	}

	if path, err := exec.LookPath("gcc"); err == nil {
		return &CCompiler{Path: path, Flavor: "gcc"}, nil
	}

	if path, err := exec.LookPath("cl"); err == nil {
		return &CCompiler{Path: path, Flavor: "cl"}, nil
	}

	return nil, fmt.Errorf("no compatible C compiler (clang, gcc, or cl) found in PATH or CC env var.\n" +
		"Please install Clang (e.g. LLVM) or GCC and add it to your PATH to run general-purpose binary tests.")
}
