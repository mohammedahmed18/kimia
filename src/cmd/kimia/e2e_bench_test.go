package main

import (
	"testing"

	"github.com/rapidfort/kimia/internal/build"
	"github.com/rapidfort/kimia/internal/validation"
	"github.com/rapidfort/kimia/pkg/logger"
)

// realisticArgs simulates a complex real-world Kimia invocation with
// multiple destinations, build-args, labels, cache options, attestations,
// insecure registries, and various flags.
func realisticArgs() []string {
	return []string{
		"--context=/workspace/myapp",
		"--context-sub-path=services/api",
		"-f", "Dockerfile.prod",
		"-d", "us-east1-docker.pkg.dev/myproject/images/api:v2.1.0",
		"-d", "gcr.io/myproject/api:v2.1.0",
		"-d", "docker.io/myorg/api:v2.1.0",
		"-d", "ghcr.io/myorg/api:v2.1.0",
		"-d", "quay.io/myorg/api:v2.1.0",
		"--build-arg", "GO_VERSION=1.25.0",
		"--build-arg", "ALPINE_VERSION=3.21",
		"--build-arg", "APP_NAME=api-server",
		"--build-arg", "APP_VERSION=2.1.0",
		"--build-arg", "BUILD_DATE=2026-04-29",
		"--build-arg", "COMMIT_SHA=abc123def456",
		"--build-arg", "BRANCH=main",
		"--build-arg", "GOPROXY=https://proxy.golang.org,direct",
		"--build-arg", "GOPRIVATE=github.com/myorg/*",
		"--build-arg", "CGO_ENABLED=0",
		"--label", "org.opencontainers.image.title=api-server",
		"--label", "org.opencontainers.image.version=2.1.0",
		"--label", "org.opencontainers.image.created=2026-04-29T12:00:00Z",
		"--label", "org.opencontainers.image.source=https://github.com/myorg/myapp",
		"--label", "org.opencontainers.image.revision=abc123def456",
		"--label", "org.opencontainers.image.vendor=MyOrg",
		"--label", "org.opencontainers.image.licenses=MIT",
		"--label", "org.opencontainers.image.description=Production API server",
		"--label", "com.myorg.team=platform",
		"--label", "com.myorg.slack-channel=platform-alerts",
		"--custom-platform", "linux/amd64",
		"-t", "production",
		"--storage-driver", "overlay",
		"--cache=true",
		"--cache-dir=/tmp/build-cache",
		"--insecure-registry", "internal-registry.myorg.local:5000",
		"--insecure-registry", "dev-registry.myorg.local:5000",
		"--push-retry", "3",
		"--image-download-retry", "2",
		"--digest-file=/tmp/digest.txt",
		"--image-name-with-digest-file=/tmp/image-digest.txt",
		"--reproducible",
		"--timestamp", "1745942400",
		"--attestation", "max",
		"-v", "info",
		"--log-timestamp",
	}
}

// BenchmarkE2EPipeline benchmarks the full orchestration pipeline:
// arg parsing -> config construction -> storage driver validation ->
// context URL detection -> validation -> command construction (both builders).
func BenchmarkE2EPipeline(b *testing.B) {
	// Suppress logger output during benchmarks
	logger.Setup("fatal", false)

	args := realisticArgs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Phase 1: Parse args into Config
		config := parseArgs(args)

		// Phase 2: Storage driver validation (from main())
		if config.StorageDriver != "" {
			validDrivers := []string{"vfs", "overlay", "native"}
			for _, driver := range validDrivers {
				if config.StorageDriver == driver {
					break
				}
			}
		}

		// Phase 3: Config to build.Config conversion (from run())
		buildConfig := build.Config{
			Dockerfile:                 config.Dockerfile,
			Destination:                config.Destination,
			Target:                     config.Target,
			BuildArgs:                  config.BuildArgs,
			Labels:                     config.Labels,
			CustomPlatform:             config.CustomPlatform,
			Cache:                      config.Cache,
			CacheDir:                   config.CacheDir,
			ExportCache:                config.ExportCache,
			ImportCache:                config.ImportCache,
			StorageDriver:              config.StorageDriver,
			Insecure:                   config.Insecure,
			InsecurePull:               config.InsecurePull,
			InsecureRegistry:           config.InsecureRegistry,
			RegistryCertificate:        config.RegistryCertificate,
			ImageDownloadRetry:         config.ImageDownloadRetry,
			NoPush:                     config.NoPush,
			TarPath:                    config.TarPath,
			DigestFile:                 config.DigestFile,
			ImageNameWithDigestFile:    config.ImageNameWithDigestFile,
			ImageNameTagWithDigestFile: config.ImageNameTagWithDigestFile,
			Reproducible:               config.Reproducible,
			Timestamp:                  config.Timestamp,
			Attestation:                config.Attestation,
			AttestationConfigs:         convertAttestationConfigs(config.AttestationConfigs),
			BuildKitOpts:               config.BuildKitOpts,
			Sign:                       config.Sign,
			CosignKeyPath:              config.CosignKeyPath,
			CosignPasswordEnv:          config.CosignPasswordEnv,
			BuildahOpts:                config.BuildahOpts,
		}

		// Phase 4: Simulate validation (the full validation pipeline)
		simulateValidation(buildConfig)

		// Phase 5: Simulate command construction for both builders
		simulateBuildKitCommand(buildConfig)
		simulateBuildahCommand(buildConfig)

		// Phase 6: sanitizeForOutput (used in error paths)
		sanitizeForOutput(config.StorageDriver, 50)
	}
}

// BenchmarkParseArgs benchmarks only argument parsing
func BenchmarkParseArgs(b *testing.B) {
	logger.Setup("fatal", false)
	args := realisticArgs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseArgs(args)
	}
}

// BenchmarkValidation benchmarks the full validation pipeline
func BenchmarkValidation(b *testing.B) {
	logger.Setup("fatal", false)
	config := parseArgs(realisticArgs())

	buildConfig := build.Config{
		Dockerfile:     config.Dockerfile,
		Destination:    config.Destination,
		Target:         config.Target,
		BuildArgs:      config.BuildArgs,
		Labels:         config.Labels,
		CustomPlatform: config.CustomPlatform,
		Reproducible:   config.Reproducible,
		Timestamp:      config.Timestamp,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		simulateValidation(buildConfig)
	}
}

// BenchmarkCommandConstruction benchmarks command construction for both builders
func BenchmarkCommandConstruction(b *testing.B) {
	logger.Setup("fatal", false)
	config := parseArgs(realisticArgs())

	buildConfig := build.Config{
		Dockerfile:       config.Dockerfile,
		Destination:      config.Destination,
		Target:           config.Target,
		BuildArgs:        config.BuildArgs,
		Labels:           config.Labels,
		CustomPlatform:   config.CustomPlatform,
		Cache:            config.Cache,
		CacheDir:         config.CacheDir,
		ExportCache:      config.ExportCache,
		ImportCache:      config.ImportCache,
		StorageDriver:    config.StorageDriver,
		Insecure:         config.Insecure,
		InsecurePull:     config.InsecurePull,
		InsecureRegistry: config.InsecureRegistry,
		Reproducible:     config.Reproducible,
		Timestamp:        config.Timestamp,
		Attestation:      config.Attestation,
		NoPush:           config.NoPush,
		ImageDownloadRetry: config.ImageDownloadRetry,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		simulateBuildKitCommand(buildConfig)
		simulateBuildahCommand(buildConfig)
	}
}

// BenchmarkSanitizeForOutput benchmarks the sanitizeForOutput function
func BenchmarkSanitizeForOutput(b *testing.B) {
	inputs := []string{
		"overlay",
		"some-driver-with-special-chars!@#$%^&*()",
		"a-very-long-storage-driver-name-that-exceeds-the-maximum-length-allowed-and-should-be-truncated-by-the-function",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, inp := range inputs {
			sanitizeForOutput(inp, 50)
		}
	}
}

// BenchmarkSanitizeCommandArgs benchmarks the command args sanitization
func BenchmarkSanitizeCommandArgs(b *testing.B) {
	logger.Setup("fatal", false)
	args := []string{
		"build", "--frontend", "dockerfile.v0",
		"--opt", "filename=Dockerfile.prod",
		"--local", "context=/workspace/myapp",
		"--opt", "build-arg:GO_VERSION=1.25.0",
		"--opt", "build-arg:ALPINE_VERSION=3.21",
		"--opt", "build-arg:APP_NAME=api-server",
		"--opt", "label:org.opencontainers.image.title=api-server",
		"--opt", "label:org.opencontainers.image.version=2.1.0",
		"--opt", "platform=linux/amd64",
		"--opt", "target=production",
		"--output", "type=image,name=gcr.io/myproject/api:v2.1.0,push=true,rewrite-timestamp=true",
		"--output", "type=image,name=docker.io/myorg/api:v2.1.0,push=true,rewrite-timestamp=true",
		"--opt", "context=https://oauth2:mytoken@github.com/myorg/myapp.git#main:services/api",
		"--opt", "build-arg:GIT_TOKEN=secret-token-value-here",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		build.SanitizeCommandArgsForBench(args)
	}
}

// simulateValidation runs all validation that would occur in the real pipeline
func simulateValidation(config build.Config) {
	// Validate destinations
	for _, dest := range config.Destination {
		validation.ValidateImageName(dest)
	}

	// Validate build args (keys and values)
	for key, value := range config.BuildArgs {
		if len(key) > 128 || len(value) > 4096 {
			continue
		}
		_ = key
		_ = value
	}

	// Validate labels
	for key, value := range config.Labels {
		if len(key) > 128 || len(value) > 4096 {
			continue
		}
		_ = key
		_ = value
	}

	// Validate platform
	if config.CustomPlatform != "" {
		validation.ValidatePlatform(config.CustomPlatform)
	}

	// Validate target
	if config.Target != "" {
		_ = config.Target
	}
}

// simulateBuildKitCommand reconstructs the buildctl command as executeBuildKit does
func simulateBuildKitCommand(config build.Config) []string {
	return build.BuildBuildKitArgs(config)
}

// simulateBuildahCommand reconstructs the buildah command as executeBuildah does
func simulateBuildahCommand(config build.Config) []string {
	return build.BuildBuildahArgs(config)
}
