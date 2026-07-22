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
	tt.Logf("Using detected C compiler: %s (%s)", cc.Path, cc.Flavor)

	testCases := []struct {
		name       string
		wuffsPath  string
		wantStdout string
	}{
		{
			name:       "gp_test_args",
			wuffsPath:  "../example/gp_test_args/main.wuffs",
			wantStdout: "",
		},
		{
			name:       "gp_test_arena",
			wuffsPath:  "../example/gp_test_arena/main.wuffs",
			wantStdout: "",
		},
		{
			name:       "gp_test_str",
			wuffsPath:  "../example/gp_test_str/main.wuffs",
			wantStdout: "",
		},
	}

	wuffsCBin := filepath.Join("..", "bin", "wuffs-c.exe")
	if _, err := os.Stat(wuffsCBin); os.IsNotExist(err) {
		wuffsCBin = filepath.Join("..", "bin", "wuffs-c")
	}

	baseCPath := filepath.Join("..", "gen", "c", "wuffs-base.c")
	baseData, err := os.ReadFile(baseCPath)
	if err != nil {
		tt.Fatalf("Failed to read gen/c/wuffs-base.c: %v", err)
	}

	for _, tc := range testCases {
		tt.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cFile := filepath.Join(tempDir, tc.name+".c")
			exeFile := filepath.Join(tempDir, tc.name+".exe")

			// Copy wuffs-base.c into tempDir so #include "./wuffs-base.c" resolves cleanly
			if err := os.WriteFile(filepath.Join(tempDir, "wuffs-base.c"), baseData, 0644); err != nil {
				t.Fatalf("Failed to copy wuffs-base.c to tempDir: %v", err)
			}

			// Transpile Wuffs source to C
			cmdGen := exec.Command(wuffsCBin, "gen", "-package_name=main", tc.wuffsPath)
			cOut, err := cmdGen.Output()
			if err != nil {
				t.Fatalf("Transpiling %s failed: %v", tc.wuffsPath, err)
			}

			// Include base headers first, define WUFFS_IMPLEMENTATION, include transpiled cOut, and append C main wrapper
			headerPrefix := []byte("#include \"./wuffs-base.c\"\n#include \"wuffs_os.h\"\n#define WUFFS_IMPLEMENTATION\n\n")
			mainWrapper := []byte("\n\nint main(int argc, char** argv) {\n" +
				"    (void)argc;\n" +
				"    (void)argv;\n" +
				"    wuffs_base__env env = wuffs_os__make_environment(argc, argv);\n" +
				"    wuffs_base__status status = wuffs_main__main(env);\n" +
				"    if (wuffs_base__status__is_error(&status)) return 1;\n" +
				"    return 0;\n" +
				"}\n")

			fullCData := append(headerPrefix, append(cOut, mainWrapper...)...)
			if err := os.WriteFile(cFile, fullCData, 0644); err != nil {
				t.Fatalf("Failed to write C file: %v", err)
			}

			var cmdCompile *exec.Cmd
			if cc.Flavor == "cl" {
				cmdCompile = exec.Command(cc.Path, "/Fe:"+exeFile, cFile, "/W4")
			} else {
				cmdCompile = exec.Command(cc.Path, "-Wall", "-Wextra", "-Wno-unused-parameter", "-Wno-unused-variable", "-Wno-unused-but-set-variable", "-Werror", "-std=c99", "-I../std/os", "-o", exeFile, cFile)
			}
			if compOut, err := cmdCompile.CombinedOutput(); err != nil {
				t.Fatalf("C compilation failed: %v\nOutput:\n%s", err, string(compOut))
			}

			cmdRun := exec.Command(exeFile)
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
