package build

import (
	"sort"
	"strconv"
	"strings"
)

// BuildBuildKitArgs constructs the buildctl command-line arguments
// the same way executeBuildKit does, without executing external commands.
// Exported for benchmarking the command-construction hot path.
func BuildBuildKitArgs(config Config) []string {
	// Pre-allocate with estimated capacity: base(3) + dockerfile(2) + context(4) +
	// buildArgs(2*N) + labels(2*N) + target(2) + platform(2) + timestamp(4) +
	// cache(1) + destinations(2*N) + attestation(4) + buildkitOpts(2*N)
	estCap := 20 + 2*len(config.BuildArgs) + 2*len(config.Labels) + 2*len(config.Destination) + 2*len(config.BuildKitOpts)
	args := make([]string, 0, estCap)
	args = append(args, "build", "--frontend", "dockerfile.v0")

	// Dockerfile
	dockerfilePath := config.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	args = append(args, "--opt", "filename="+dockerfilePath)

	// Context (local path simulation)
	buildContext := "/workspace"
	args = append(args, "--local", "context="+buildContext)
	args = append(args, "--local", "dockerfile="+buildContext)

	// Sorted build args
	buildArgKeys := make([]string, 0, len(config.BuildArgs))
	for key := range config.BuildArgs {
		buildArgKeys = append(buildArgKeys, key)
	}
	sort.Strings(buildArgKeys)

	for _, key := range buildArgKeys {
		value := config.BuildArgs[key]
		if value != "" {
			args = append(args, "--opt", "build-arg:"+key+"="+value)
		} else {
			args = append(args, "--opt", "build-arg:"+key)
		}
	}

	// Sorted labels
	labelKeys := make([]string, 0, len(config.Labels))
	for key := range config.Labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)

	for _, key := range labelKeys {
		value := config.Labels[key]
		args = append(args, "--opt", "label:"+key+"="+value)
	}

	// Target
	if config.Target != "" {
		args = append(args, "--opt", "target="+config.Target)
	}

	// Platform
	if config.CustomPlatform != "" {
		args = append(args, "--opt", "platform="+config.CustomPlatform)
	}

	// Reproducible timestamp
	var sourceEpoch string
	if config.Reproducible && config.Timestamp != "" {
		sourceEpoch = config.Timestamp
		args = append(args, "--opt", "source-date-epoch="+sourceEpoch)
		args = append(args, "--opt", "build-arg:SOURCE_DATE_EPOCH="+sourceEpoch)
	}

	// Cache control
	if !config.Cache || config.Reproducible {
		args = append(args, "--no-cache")
	}

	// Cache import/export
	for _, ic := range config.ImportCache {
		if !config.Reproducible {
			args = append(args, "--import-cache", ic)
		}
	}
	for _, ec := range config.ExportCache {
		if !config.Reproducible {
			args = append(args, "--export-cache", ec)
		}
	}

	// Sorted destinations + output
	sortedDests := make([]string, len(config.Destination))
	copy(sortedDests, config.Destination)
	sort.Strings(sortedDests)

	if config.TarPath != "" {
		outputOpts := "type=docker,dest=" + config.TarPath
		if config.Reproducible && sourceEpoch != "" {
			outputOpts += ",rewrite-timestamp=true"
		}
		args = append(args, "--output", outputOpts)
	} else if !config.NoPush {
		for _, dest := range sortedDests {
			outputOpts := "type=image,name=" + dest + ",push=true"
			if config.Reproducible && sourceEpoch != "" {
				outputOpts += ",rewrite-timestamp=true"
			}
			args = append(args, "--output", outputOpts)
		}
	} else {
		for _, dest := range sortedDests {
			outputOpts := "type=image,name=" + dest + ",push=false"
			if config.Reproducible && sourceEpoch != "" {
				outputOpts += ",rewrite-timestamp=true"
			}
			args = append(args, "--output", outputOpts)
		}
	}

	// Attestation
	if config.Attestation != "off" && config.Attestation != "" {
		reproducibleSuffix := ""
		if config.Reproducible {
			reproducibleSuffix = ",reproducible=true"
		}
		switch config.Attestation {
		case "min":
			args = append(args, "--opt", "attest:sbom=false")
			args = append(args, "--opt", "attest:provenance=mode=min"+reproducibleSuffix)
		case "max":
			args = append(args, "--opt", "attest:sbom=true")
			args = append(args, "--opt", "attest:provenance=mode=max"+reproducibleSuffix)
		}
	}

	// BuildKit pass-through opts
	for _, opt := range config.BuildKitOpts {
		args = append(args, "--opt", opt)
	}

	return args
}

// BuildBuildahArgs constructs the buildah bud command-line arguments
// the same way executeBuildah does, without executing external commands.
// Exported for benchmarking the command-construction hot path.
func BuildBuildahArgs(config Config) []string {
	// Pre-allocate: base(1) + dockerfile(2) + buildArgs(2*N) + labels(2*N) +
	// target(2) + platform(2) + cache(1) + retry(2) + timestamp(2) + insecure(1) +
	// destinations(2*N) + buildahOpts(2*N) + context(1)
	estCap := 14 + 2*len(config.BuildArgs) + 2*len(config.Labels) + 2*len(config.Destination) + 2*len(config.BuildahOpts)
	args := make([]string, 0, estCap)
	args = append(args, "bud")

	// Dockerfile
	dockerfilePath := config.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	args = append(args, "-f", dockerfilePath)

	// Sorted build args
	buildArgKeys := make([]string, 0, len(config.BuildArgs))
	for key := range config.BuildArgs {
		buildArgKeys = append(buildArgKeys, key)
	}
	sort.Strings(buildArgKeys)

	for _, key := range buildArgKeys {
		value := config.BuildArgs[key]
		if value != "" {
			args = append(args, "--build-arg", key+"="+value)
		} else {
			args = append(args, "--build-arg", key)
		}
	}

	// Sorted labels
	labelKeys := make([]string, 0, len(config.Labels))
	for key := range config.Labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)

	for _, key := range labelKeys {
		value := config.Labels[key]
		args = append(args, "--label", key+"="+value)
	}

	// Target
	if config.Target != "" {
		args = append(args, "--target", config.Target)
	}

	// Platform
	if config.CustomPlatform != "" {
		args = append(args, "--platform", config.CustomPlatform)
	}

	// Cache
	if config.Cache && !config.Reproducible {
		args = append(args, "--layers")
	} else {
		args = append(args, "--no-cache")
	}

	// Retry
	if config.ImageDownloadRetry > 0 {
		args = append(args, "--retry", strconv.Itoa(config.ImageDownloadRetry))
	}

	// Reproducible timestamp
	if config.Reproducible && config.Timestamp != "" {
		args = append(args, "--timestamp", config.Timestamp)
	}

	// Insecure
	if config.Insecure || config.InsecurePull {
		args = append(args, "--tls-verify=false")
	}

	// Sorted destinations
	sortedDests := make([]string, len(config.Destination))
	copy(sortedDests, config.Destination)
	sort.Strings(sortedDests)

	for _, dest := range sortedDests {
		args = append(args, "-t", dest)
	}

	// Buildah pass-through opts
	for _, opt := range config.BuildahOpts {
		parts := strings.SplitN(opt, " ", 2)
		if len(parts) == 2 {
			args = append(args, parts[0], parts[1])
		} else {
			args = append(args, parts[0])
		}
	}

	// Context path
	args = append(args, "/workspace")

	return args
}

// SanitizeCommandArgsForBench exposes sanitizeCommandArgs for benchmarking.
func SanitizeCommandArgsForBench(args []string) []string {
	return sanitizeCommandArgs(args)
}
