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
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/wuffs/internal/cgen"
)

type buildConfig struct {
	input        string
	output       string
	compiler     string
	optimization string
	debug        bool
	verbose      bool
}

type buildResult struct {
	executable string
	projectDir string
}

type cCompiler struct {
	path   string
	flavor string
}

func doBuild(wuffsRoot string, args []string) error {
	result, err := buildApplication(wuffsRoot, args)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	fmt.Println("build:", result.executable)
	return nil
}

func doRun(wuffsRoot string, args []string) error {
	buildArgs, programArgs := splitRunArgs(args)
	result, err := buildApplication(wuffsRoot, buildArgs)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	cmd := exec.Command(result.executable, programArgs...)
	cmd.Dir = result.projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", result.executable, err)
	}
	return nil
}

func splitRunArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func parseBuildConfig(args []string) (buildConfig, error) {
	fs := flag.NewFlagSet("wuffs build", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: wuffs build [flags] [directory|file.wuffs]")
		fs.PrintDefaults()
	}
	output := fs.String("o", "", "output executable")
	compiler := fs.String("cc", "", "C compiler (defaults to CC, clang, gcc, or cl)")
	optimization := fs.String("O", "2", "C compiler optimization level")
	debug := fs.Bool("debug", false, "include debug information and disable optimization")
	verbose := fs.Bool("v", false, "print the generated and compiler commands")
	if err := fs.Parse(args); err != nil {
		return buildConfig{}, err
	}
	if len(fs.Args()) > 1 {
		return buildConfig{}, fmt.Errorf("wuffs build accepts at most one source directory or file")
	}
	input := "."
	if len(fs.Args()) == 1 {
		input = fs.Args()[0]
	}
	opt := strings.TrimPrefix(*optimization, "-")
	if opt == "" || strings.ContainsAny(opt, " \t\r\n") {
		return buildConfig{}, fmt.Errorf("invalid -O value %q", *optimization)
	}
	return buildConfig{
		input:        input,
		output:       *output,
		compiler:     *compiler,
		optimization: opt,
		debug:        *debug,
		verbose:      *verbose,
	}, nil
}

func buildApplication(wuffsRoot string, args []string) (buildResult, error) {
	cfg, err := parseBuildConfig(args)
	if err != nil {
		return buildResult{}, err
	}
	projectDir, sources, err := applicationSources(cfg.input)
	if err != nil {
		return buildResult{}, err
	}
	compiler, err := findCCompiler(cfg.compiler)
	if err != nil {
		return buildResult{}, err
	}

	output := cfg.output
	if output == "" {
		output = filepath.Join(projectDir, "build", "main"+executableSuffix())
	} else if !filepath.IsAbs(output) {
		cwd, err := os.Getwd()
		if err != nil {
			return buildResult{}, err
		}
		output = filepath.Join(cwd, output)
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return buildResult{}, err
	}

	key, err := buildKey(wuffsRoot, projectDir, compiler, cfg)
	if err != nil {
		return buildResult{}, err
	}
	stageDir := filepath.Join(projectDir, "build", ".wuffs", key)
	mainC := filepath.Join(stageDir, "gen", "c", "wuffs-main.c")
	if fileExists(output) && fileExists(mainC) {
		return buildResult{executable: output, projectDir: projectDir}, nil
	}

	if err := os.MkdirAll(filepath.Join(stageDir, "gen", "c"), 0755); err != nil {
		return buildResult{}, err
	}
	osHeader := []byte(cgen.EmbeddedString_OSHeader)
	if wuffsRoot != "" {
		data, err := os.ReadFile(filepath.Join(wuffsRoot, "std", "os", "wuffs_os.h"))
		if err != nil {
			return buildResult{}, fmt.Errorf("reading Wuffs OS runtime: %w", err)
		}
		osHeader = data
	}
	if err := writeFileBytes(filepath.Join(stageDir, "std", "os", "wuffs_os.h"), osHeader); err != nil {
		return buildResult{}, fmt.Errorf("copying Wuffs OS runtime: %w", err)
	}

	resolver := buildUseResolver(wuffsRoot, projectDir)
	h := genHelper{
		wuffsRoot:  wuffsRoot,
		outputRoot: stageDir,
		sourceRoot: projectDir,
		langs:      []string{"c"},
		resolveUse: resolver,
		quiet:      true,
	}
	if err := h.genDirDependencies(sources); err != nil {
		return buildResult{}, err
	}

	cgenArgs := []string{"-package_name=main"}
	cgenArgs = append(cgenArgs, sources...)
	generated, err := cgen.GenerateWithResolver(cgenArgs, resolver)
	if err != nil {
		return buildResult{}, err
	}
	if err := os.WriteFile(mainC, generated, 0644); err != nil {
		return buildResult{}, err
	}

	if cfg.verbose {
		fmt.Printf("wuffs-c gen -package_name=main %s\n", strings.Join(sources, " "))
	}
	if err := compileApplication(compiler, cfg, projectDir, stageDir, mainC, output); err != nil {
		return buildResult{}, err
	}
	return buildResult{executable: output, projectDir: projectDir}, nil
}

func applicationSources(input string) (string, []string, error) {
	input, err := filepath.Abs(input)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(input)
	if err != nil {
		return "", nil, err
	}
	if !info.IsDir() {
		if filepath.Ext(input) != ".wuffs" {
			return "", nil, fmt.Errorf("build input %q is not a Wuffs source file", input)
		}
		return filepath.Dir(input), []string{input}, nil
	}

	entries, err := os.ReadDir(input)
	if err != nil {
		return "", nil, err
	}
	sources := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".wuffs" {
			sources = append(sources, filepath.Join(input, entry.Name()))
		}
	}
	sort.Strings(sources)
	if len(sources) == 0 {
		return "", nil, fmt.Errorf("no .wuffs files found in %q", input)
	}
	return input, sources, nil
}

func findCCompiler(request string) (cCompiler, error) {
	candidates := []string(nil)
	if request != "" {
		candidates = append(candidates, request)
	} else if cc := os.Getenv("CC"); cc != "" {
		candidates = append(candidates, cc)
	} else {
		candidates = []string{"clang", "gcc", "cl"}
	}

	for _, candidate := range candidates {
		if strings.ContainsAny(candidate, " \t\r\n") {
			return cCompiler{}, fmt.Errorf("C compiler %q contains arguments; use -cc for the executable and keep flags in the build configuration", candidate)
		}
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		base := strings.ToLower(filepath.Base(path))
		flavor := "gcc"
		if strings.Contains(base, "clang") {
			flavor = "clang"
		} else if base == "cl" || base == "cl.exe" {
			flavor = "cl"
		}
		return cCompiler{path: path, flavor: flavor}, nil
	}
	return cCompiler{}, fmt.Errorf("no C compiler found; install clang or gcc, or set CC / use -cc")
}

func compileApplication(compiler cCompiler, cfg buildConfig, projectDir, stageDir, source, output string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	args := []string(nil)
	if compiler.flavor == "cl" {
		args = append(args, "/nologo", "/W4")
		if cfg.debug {
			args = append(args, "/Od", "/Zi")
		} else {
			args = append(args, "/O2")
		}
		args = append(args, "/I"+stageDir, "/I"+projectDir, "/Fe:"+output, source)
	} else {
		if cfg.debug {
			args = append(args, "-O0", "-g")
		} else {
			args = append(args, "-O"+cfg.optimization)
		}
		args = append([]string{"-std=c99"}, args...)
		args = append(args, "-I", stageDir, "-I", projectDir, "-o", output, source)
	}
	if cfg.verbose {
		fmt.Printf("%s %s\n", compiler.path, strings.Join(args, " "))
	}
	cmd := exec.Command(compiler.path, args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("C compiler failed: %w", err)
	}
	return nil
}

func buildKey(wuffsRoot, projectDir string, compiler cCompiler, cfg buildConfig) (string, error) {
	h := sha256.New()
	h.Write([]byte("wuffs-build-v1\n"))
	h.Write([]byte(compiler.path + "\n" + compiler.flavor + "\n" + cfg.optimization + "\n"))
	if version, err := exec.Command(compiler.path, "--version").CombinedOutput(); err == nil {
		h.Write(version)
	}
	if cfg.debug {
		h.Write([]byte("debug\n"))
	}
	if err := hashProjectWuffsSources(h, projectDir); err != nil {
		return "", err
	}
	if manifest := filepath.Join(projectDir, "wuffs.mod"); fileExists(manifest) {
		if err := hashFile(h, projectDir, manifest); err != nil {
			return "", err
		}
	}
	if err := hashWuffsSources(h, wuffsRoot); err != nil {
		return "", err
	}
	if exe, err := os.Executable(); err == nil {
		if err := hashFile(h, "toolchain", exe); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:24], nil
}

func hashProjectWuffsSources(h interface{ Write([]byte) (int, error) }, projectDir string) error {
	var filenames []string
	err := filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != projectDir && (entry.Name() == "build" || entry.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) == ".wuffs" {
			filenames = append(filenames, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		if err := hashFile(h, projectDir, filename); err != nil {
			return err
		}
	}
	return nil
}

func hashFile(h interface{ Write([]byte) (int, error) }, root, filename string) error {
	rel := filename
	if absRoot, err := filepath.Abs(root); err == nil {
		if candidate, err := filepath.Rel(absRoot, filename); err == nil {
			rel = candidate
		}
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	h.Write([]byte(filepath.ToSlash(rel) + "\x00"))
	h.Write(data)
	h.Write([]byte("\x00"))
	return nil
}

func hashWuffsSources(h interface{ Write([]byte) (int, error) }, wuffsRoot string) error {
	if wuffsRoot == "" {
		return nil
	}
	for _, dirname := range []string{"std", filepath.Join("gen", "wuffs")} {
		root := filepath.Join(wuffsRoot, dirname)
		if !fileExists(root) {
			continue
		}
		var filenames []string
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".wuffs" {
				filenames = append(filenames, path)
			}
			return nil
		})
		if err != nil {
			return err
		}
		sort.Strings(filenames)
		for _, filename := range filenames {
			if err := hashFile(h, wuffsRoot, filename); err != nil {
				return err
			}
		}
	}
	return hashFile(h, wuffsRoot, filepath.Join(wuffsRoot, "std", "os", "wuffs_os.h"))
}

func buildUseResolver(wuffsRoot, projectDir string) func(string) ([]byte, error) {
	return func(usePath string) ([]byte, error) {
		rel := filepath.FromSlash(usePath)
		candidates := []string{
			filepath.Join(projectDir, rel),
		}
		if wuffsRoot != "" {
			candidates = append(candidates,
				filepath.Join(wuffsRoot, "gen", "wuffs", rel),
				filepath.Join(wuffsRoot, rel),
			)
		}
		for _, candidate := range candidates {
			if data, err := os.ReadFile(candidate); err == nil {
				return data, nil
			}
			if data, err := readWuffsDirectory(strings.TrimSuffix(candidate, ".wuffs")); err == nil {
				return data, nil
			}
		}
		return nil, fmt.Errorf("could not resolve use %q", strings.TrimSuffix(usePath, ".wuffs"))
	}
}

func readWuffsDirectory(dirname string) ([]byte, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var out []byte
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wuffs" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dirname, entry.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, data...)
		out = append(out, '\n')
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no Wuffs files in %q", dirname)
	}
	return out, nil
}

func writeFileBytes(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func executableSuffix() string {
	if filepath.Separator == '\\' {
		return ".exe"
	}
	return ""
}

func doInit(_ string, args []string) error {
	fs := flag.NewFlagSet("wuffs init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("wuffs init accepts at most one module name")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	moduleName := filepath.Base(cwd)
	if len(fs.Args()) == 1 {
		moduleName = fs.Args()[0]
	}
	if moduleName == "" || strings.ContainsAny(moduleName, " \t\r\n") {
		return fmt.Errorf("invalid module name %q", moduleName)
	}
	manifest := filepath.Join(cwd, "wuffs.mod")
	if fileExists(manifest) {
		return fmt.Errorf("wuffs.mod already exists")
	}
	if err := os.WriteFile(manifest, []byte("module "+moduleName+"\n"), 0644); err != nil {
		return err
	}
	gitignore := filepath.Join(cwd, ".gitignore")
	if !fileExists(gitignore) {
		if err := os.WriteFile(gitignore, []byte("build/\n"), 0644); err != nil {
			return err
		}
	}
	fmt.Println("initialized Wuffs module", moduleName)
	return nil
}

func doClean(_ string, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("wuffs clean accepts at most one directory")
	}
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("clean target %q is not a directory", dir)
	}
	cacheDir := filepath.Join(dir, "build", ".wuffs")
	if err := os.RemoveAll(cacheDir); err != nil {
		return err
	}
	for _, filename := range []string{
		filepath.Join(dir, "build", "main"+executableSuffix()),
		filepath.Join(dir, "build", "main"),
	} {
		if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	fmt.Println("cleaned:", filepath.Join(dir, "build"))
	return nil
}
