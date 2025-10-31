# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- **Config.Sanitize()** method removed - resilience config contains no secrets
  - This was implementing the method unnecessarily for "consistency"
  - Module now only implements `Prefix()` and `ConfigSummary()`
  - No breaking change for users who weren't calling `Sanitize()` directly

## [0.2.0] - 2025-10-29

### Added
- Release version 0.2.0

### Changed
- Updated gostratum dependencies to latest versions


## [0.1.5] - 2025-10-28

### Added

- Enhance Makefile and scripts for version management and dependency updates

## [0.1.4] - 2025-10-27

### Changed

- Update gostratum/core dependency to v0.1.8

### Added

- Update NewConfig to return sanitized Config copy
- Add Sanitize and ConfigSummary methods to Config struct

### Changed

- Format code for consistency in test files

## [0.1.3] - 2025-10-26

### Changed

- Update gostratum/core dependency to v0.1.7

### Changed

- Change return type from interface{} to any in executor methods

## [0.1.2] - 2025-10-25

### Changed

- Update gostratum/core dependency to v0.1.5

### Fixed

- Update gostratum/core dependency to v0.1.4

### Added

- Add unit tests for bulkhead and resilience configurations

## [0.1.1] - 2025-10-24

### Added

- Add unit tests for circuit breaker, rate limiter, and retry mechanisms

## [0.1.0] - 2025-10-23

### Added

- Implement resilience module with circuit breaker, retry, rate limiter, bulkhead, and timeout patterns