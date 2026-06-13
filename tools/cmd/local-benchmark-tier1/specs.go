package main

import (
	"os"
	"path/filepath"
)

func buildBenchmarkSpecs(outDir string) []benchmarkSpec {
	var specs []benchmarkSpec
	for _, category := range requiredP20Categories {
		for _, language := range requiredLanguages {
			name := slug(category) + "_" + language
			spec := benchmarkSpec{
				Name:             name,
				Category:         category,
				Language:         language,
				AlgorithmID:      "p25.0." + slug(category),
				InputDescription: inputDescription(category),
				SourceRelPath:    filepath.Join(outDir, "artifacts", "src", name+extensionFor(language)),
				BinaryRelPath:    filepath.Join(outDir, "artifacts", "bin", name),
			}
			switch language {
			case "tetra":
				spec.BuildCommandKind = "tetra"
				spec.BuildArgs = []string{"tetra", "build", "--target", "linux-x64", "--explain"}
				if category != "actor ping-pong" {
					spec.SourceRelPath = filepath.Join(outDir, "artifacts", "src", "p25", slug(category)+".tetra")
				}
				spec.Source = tetraSource(category)
			case "c":
				spec.BuildCommandKind = "clang"
				spec.BuildArgs = []string{"clang", "-O3"}
				spec.Source = cLikeSource(category)
			case "cpp":
				spec.BuildCommandKind = "clang++"
				spec.BuildArgs = []string{"clang++", "-O3"}
				spec.Source = cLikeSource(category)
			case "rust":
				spec.BuildCommandKind = "rustc"
				spec.BuildArgs = []string{"rustc", "-C", "opt-level=3"}
				spec.Source = rustSource(category)
			}
			specs = append(specs, spec)
		}
	}
	return specs
}

func executeSpec(spec benchmarkSpec, opt options, env []string, versions map[string]string, tetraTool string, optimizerArtifact string) benchmarkRow {
	sourcePath := spec.SourceRelPath
	binaryPath := spec.BinaryRelPath
	buildStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stdout.txt")
	buildStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stderr.txt")
	runStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stdout.txt")
	runStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stderr.txt")
	_ = os.MkdirAll(filepath.Dir(sourcePath), 0o755)
	_ = os.WriteFile(sourcePath, []byte(spec.Source), 0o644)

	buildCommand := buildCommand(spec, tetraTool)
	runCommand := []string{binaryPath}
	row := benchmarkRow{
		Name:            spec.Name,
		Category:        spec.Category,
		Language:        spec.Language,
		Status:          "measured",
		CompilerVersion: versions[spec.Language],
		BuildCommand:    buildCommand,
		RunCommand:      runCommand,
		SourcePath:      sourcePath,
		BinaryPath:      binaryPath,
		RawOutputArtifacts: []string{
			buildStdout,
			buildStderr,
			runStdout,
			runStderr,
		},
	}
	_, buildDuration, err := runCaptured(opt.Timeout, buildCommand, env, buildStdout, buildStderr)
	row.CompileTimeMS = millis(buildDuration)
	if err != nil {
		row.Status = "build_failed"
		row.Error = err.Error()
		ensureRawRunArtifacts(runStdout, runStderr, "not run because build failed\n")
		if spec.Language == "tetra" {
			row.TetraMetadata = missingTetraMetadata(binaryPath, optimizerArtifact)
		}
		return row
	}
	if info, err := os.Stat(binaryPath); err == nil {
		row.BinarySizeBytes = info.Size()
	}
	measurements, runErr := runIterations(opt.Timeout, runCommand, env, opt.Iterations, runStdout, runStderr)
	row.RunMeasurementsMS = measurements
	row.MedianRuntimeMS = median(measurements)
	if runErr != nil {
		row.Status = "run_failed"
		row.Error = runErr.Error()
	}
	if spec.Language == "tetra" {
		row.TetraMetadata = collectTetraMetadata(spec.Name, binaryPath, optimizerArtifact)
	}
	return row
}

func buildCommand(spec benchmarkSpec, tetraTool string) []string {
	switch spec.Language {
	case "tetra":
		return []string{tetraTool, "build", "--target", "linux-x64", "--explain", "-o", spec.BinaryRelPath, spec.SourceRelPath}
	case "c":
		return []string{"clang", "-O3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	case "cpp":
		return []string{"clang++", "-O3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	case "rust":
		return []string{"rustc", "-C", "opt-level=3", spec.SourceRelPath, "-o", spec.BinaryRelPath}
	default:
		return []string{spec.BuildCommandKind}
	}
}
