// Copyright 2026 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/wuffs/internal/testcut"
)

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestGeneralPurposeApplications(tt *testing.T) {
	cc, err := testcut.FindCCompiler()
	if err != nil {
		tt.Skipf("Skipping GP application execution test: %v", err)
		return
	}
	hasSanitizers := cc.HasSanitizerSupport(tt.TempDir())
	tt.Logf("Using detected C compiler: %s (%s, ASan/UBSan: %v)", cc.Path, cc.Flavor, hasSanitizers)

	testCases := []struct {
		name       string
		wuffsPath  string
		args       []string
		wantStdout string
	}{
		{
			name:       "gp_test_args",
			wuffsPath:  "../example/gp_test_args/main.wuffs",
			args:       []string{"hello-arg"},
			wantStdout: "hello-arg\n",
		},
		{
			name:       "gp_test_arena",
			wuffsPath:  "../example/gp_test_arena/main.wuffs",
			wantStdout: "Arena test passed\n",
		},
		{
			name:       "gp_test_str",
			wuffsPath:  "../example/gp_test_str/main.wuffs",
			wantStdout: "Hello Wuffs GP\n",
		},
		{
			name:       "gp_test_packages",
			wuffsPath:  "../example/gp_test_packages/main.wuffs",
			wantStdout: "Packages test passed\n",
		},
		{
			name:       "gp_test_ffi",
			wuffsPath:  "../example/gp_test_ffi/main.wuffs",
			wantStdout: "Hello from C FFI!\n",
		},
		{
			name:       "imageinfo",
			wuffsPath:  "../example/imageinfo/main.wuffs",
			wantStdout: "Imageinfo initialized\n",
		},
	}

	wuffsCBin := filepath.Join("..", "bin", "wuffs-c.exe")
	if _, err := os.Stat(wuffsCBin); os.IsNotExist(err) {
		wuffsCBin = filepath.Join("..", "bin", "wuffs-c")
	}

	for _, tc := range testCases {
		tt.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cFile := filepath.Join(tempDir, tc.name+".c")
			exeFile := filepath.Join(tempDir, tc.name+".exe")

			// Copy all generated C files from gen/c into tempDir
			if genCDir, err := os.ReadDir(filepath.Join("..", "gen", "c")); err == nil {
				for _, entry := range genCDir {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".c") {
						data, err := os.ReadFile(filepath.Join("..", "gen", "c", entry.Name()))
						if err == nil {
							_ = os.WriteFile(filepath.Join(tempDir, entry.Name()), data, 0644)
						}
					}
				}
			}

			// Transpile Wuffs source to C
			cmdGen := exec.Command(wuffsCBin, "gen", "-package_name=main", tc.wuffsPath)
			cOut, err := cmdGen.Output()
			if err != nil {
				t.Fatalf("Transpiling %s failed: %v", tc.wuffsPath, err)
			}

			if err := os.WriteFile(cFile, cOut, 0644); err != nil {
				t.Fatalf("Failed to write C file: %v", err)
			}

			var cmdCompile *exec.Cmd
			if cc.Flavor == "cl" {
				cmdCompile = exec.Command(cc.Path, "/Fe:"+exeFile, cFile, "/W4")
			} else {
				flags := []string{"-Wall", "-Wextra", "-Wno-unused-parameter", "-Wno-unused-variable", "-Wno-unused-but-set-variable", "-Wno-unused-function", "-Werror", "-std=c99", "-I..", "-I../std/os"}
				if hasSanitizers {
					flags = append(flags, "-fsanitize=address,undefined")
				}
				flags = append(flags, "-o", exeFile, cFile)
				cmdCompile = exec.Command(cc.Path, flags...)
			}
			if compOut, err := cmdCompile.CombinedOutput(); err != nil {
				t.Fatalf("C compilation failed: %v\nOutput:\n%s", err, string(compOut))
			}

			cmdRun := exec.Command(exeFile, tc.args...)
			runOut, err := cmdRun.CombinedOutput()
			if err != nil {
				t.Fatalf("Execution failed: %v\nOutput:\n%s", err, string(runOut))
			}

			got := normalizeNewlines(string(runOut))
			want := normalizeNewlines(tc.wantStdout)
			if got != want {
				t.Errorf("stdout mismatch:\ngot:\n%q\nwant:\n%q", got, want)
			}
		})
	}
}
