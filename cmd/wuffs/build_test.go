// Copyright 2026 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAndRunApplication(t *testing.T) {
	if _, err := findCCompiler(""); err != nil {
		t.Skip(err)
	}

	wuffsRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	source := `use "std/crc32"

pub func main!(env: base.env) base.status {
    var hasher : crc32.ieee_hasher
    var sum    : base.u32
    sum = hasher.update_u32!(x: utility.empty_slice_u8())
    args.env.print!(s: "build works\n")
    return ok
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.wuffs"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(projectDir, "build", "hello"+executableSuffix())
	result, err := buildApplication(wuffsRoot, []string{"-o", output, projectDir})
	if err != nil {
		t.Fatal(err)
	}
	if result.executable != output {
		t.Fatalf("executable = %q, want %q", result.executable, output)
	}

	got, err := exec.Command(result.executable).CombinedOutput()
	if err != nil {
		t.Fatalf("running generated application: %v\n%s", err, got)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != "build works\n" {
		t.Fatalf("output = %q, want %q", got, "build works\n")
	}

	stageRoot := filepath.Join(projectDir, "build", ".wuffs")
	entries, err := os.ReadDir(stageRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("cache entries = %d, want 1", len(entries))
	}
	stageDir := filepath.Join(stageRoot, entries[0].Name())
	for _, filename := range []string{
		filepath.Join(stageDir, "gen", "c", "wuffs-base.c"),
		filepath.Join(stageDir, "gen", "c", "wuffs-std-crc32.c"),
		filepath.Join(stageDir, "std", "os", "wuffs_os.h"),
	} {
		if _, err := os.Stat(filename); err != nil {
			t.Fatalf("staged file %q: %v", filename, err)
		}
	}
}

func TestBuildWithoutWuffsRoot(t *testing.T) {
	if _, err := findCCompiler(""); err != nil {
		t.Skip(err)
	}
	projectDir := t.TempDir()
	source := `pub func main!(env: base.env) base.status {
    args.env.print!(s: "rootless build works\n")
    return ok
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.wuffs"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(projectDir, "build", "hello"+executableSuffix())
	result, err := buildApplication("", []string{"-o", output, projectDir})
	if err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(result.executable).CombinedOutput()
	if err != nil {
		t.Fatalf("running generated application: %v\n%s", err, got)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != "rootless build works\n" {
		t.Fatalf("output = %q, want %q", got, "rootless build works\n")
	}
}
