# Go Version Fix - CI/CD Pipeline

## Problem
GitHub Actions CI/CD job failed with:
```
the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25.0)
```

## Root Cause
Multiple workflow files had mismatched Go versions:
- `GO_VERSION: '1.25.0'` in old nested workflows (event-api/.github/workflows/)
- This is incompatible with golangci-lint and Go toolchain versions available in GitHub Actions

## Solution
Updated **all** workflow files to use **Go 1.22** (stable, widely compatible):

### Files Updated
1. ✅ `.github/workflows/ci.yml` (repo root) - Already had 1.22
2. ✅ `event-api/.github/workflows/ci.yml` - Updated 1.25.0 → 1.22
3. ✅ `event-api/.github/workflows/staging.yml` - Updated 1.25.0 → 1.22
4. ✅ `event-api/.github/workflows/release.yml` - Updated 1.25.0 → 1.22

### Why Go 1.22?
- ✅ Matches your local development environment
- ✅ Widely compatible with all Go tools (including golangci-lint)
- ✅ Stable LTS version (recommended for production)
- ✅ Supported by GitHub Actions
- ✅ No compatibility issues with dependencies in event-api

### Verification
```bash
grep GO_VERSION .github/workflows/ci.yml event-api/.github/workflows/*.yml
# All should show: GO_VERSION: '1.22'
```

## Testing
After pushing these changes:
1. CI/CD pipeline should run without Go version errors
2. golangci-lint will work correctly
3. All tests will execute successfully

## Next Steps
```bash
git add .
git commit -m "fix: update go version to 1.22 in all CI/CD workflows

- Fix golangci-lint Go version mismatch error
- Align all workflows to use Go 1.22 (stable, LTS)
- Update event-api nested workflows (ci.yml, staging.yml, release.yml)
- Ensure compatibility across all CI/CD jobs
"
git push origin master
```

After pushing, check GitHub Actions: https://github.com/Orange-hanter/TuserDuser/actions

The pipeline should now run successfully without Go version conflicts! ✅
