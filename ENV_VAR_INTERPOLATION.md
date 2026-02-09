# Environment Variable Interpolation - Complete Fix

## Problem Summary
The config.yaml file referenced environment variables using `${VAR_NAME}` syntax, but these were not being interpolated when building the agent. The literal string `${SUPABASE_DATABASE_URL}` was being passed to the postgres tool, causing connection failures.

## Root Cause
There were TWO issues:

1. **Missing interpolation in config loading** - The YAML file was parsed without replacing environment variable placeholders
2. **Agent builder bypassing interpolated config** - Even after adding interpolation, the agent builder was re-reading the config file directly instead of using the already-interpolated config from ConfigManager

## Complete Solution

### 1. Added Environment Variable Interpolation to Config Loading
**File: `cmd/app/config_manager.go`**

- Added `interpolateEnvVars()` function that replaces `${VAR_NAME}` patterns with actual environment variable values
- Modified `Load()` method to interpolate before YAML unmarshaling
- Returns error if referenced environment variable is not set

**File: `cmd/app/main.go`**

- Added `godotenv.Load()` at startup to load `.env` file before config parsing

### 2. Fixed Agent Builder to Use Interpolated Config
**File: `src/builder/agentBuilder.go`**

- Added `NewAgentBuilderFromConfigStruct()` - accepts pre-processed Config struct
- Refactored `NewAgentBuilderFromConfig()` to use the new function internally
- This allows using already-interpolated config instead of re-reading from file

**File: `src/builder/vectorBuilder.go`**

- Added `NewVectorBuilderFromConfigStruct()` - accepts pre-processed VectorStorageConfig
- Ensures vector database config also gets interpolation support

**File: `cmd/app/agent_manager.go`**

- Modified `buildAgent()` to use `NewAgentBuilderFromConfigStruct()` with ConfigManager's interpolated config
- Prevents re-reading config file and bypassing interpolation

### 3. Comprehensive Tests
**File: `cmd/app/config_manager_test.go`**

- Unit tests for `interpolateEnvVars()` function
- Integration test for full config loading with interpolation
- Tests for edge cases (missing vars, special characters, multiple vars)

## How It Works Now

1. Application starts → `godotenv.Load()` loads `.env` file
2. ConfigManager reads YAML → calls `interpolateEnvVars()` → replaces all `${VAR_NAME}` patterns
3. YAML is parsed with actual values
4. Agent builder uses the interpolated Config struct (doesn't re-read file)
5. Postgres tool receives actual connection string

## Configuration Example

**config.yaml:**
```yaml
agent:
  tools:
    - name: postgres
      postgresURL: "${SUPABASE_DATABASE_URL}"
      mode: "read"
```

**.env:**
```
SUPABASE_DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
```

**Result after interpolation:**
```yaml
agent:
  tools:
    - name: postgres
      postgresURL: "postgres://user:pass@host:5432/db?sslmode=require"
      mode: "read"
```

## Verification

Check the loaded config:
```bash
curl -s http://localhost:8080/api/config | python3 -m json.tool | grep -A5 postgres
```

Should show the fully interpolated database URL, not the `${SUPABASE_DATABASE_URL}` placeholder.

## Error Handling

If an environment variable is referenced but not set:
```
config error: interpolate env vars: environment variable not set: MISSING_VAR
```

This catches configuration issues early during application startup.

## Testing

Run tests:
```bash
go test ./cmd/app/... -v
go test ./src/builder/... -v
```

All tests should pass, including the new interpolation tests.
