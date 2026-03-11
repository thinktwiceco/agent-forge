# API Tool

A unified, configuration-driven HTTP client tool. Add a new API service by dropping a JSON file in a folder — no code changes required.

## Features

- **Multi-service**: One tool instance handles any number of API services
- **Progressive discovery**: Agent lists services, then endpoints, then calls — reducing context noise
- **URL parameter substitution**: Path parameters like `/users/{user_id}`
- **Query parameters**: Key-value object → query string automatically
- **Body serialization**: Key-value object → JSON or form-encoded, based on endpoint config
- **Basic Auth shorthand**: Provide `basic_auth` and the tool computes the `Authorization: Basic` header
- **`${ENV_VAR}` expansion**: Works in header values and URLs, resolved at call time
- **Resolver functions**: Body arguments can be transparently transformed before the request (e.g. local file → base64 data URI)

## Configuration

### `config.yaml`

```yaml
agent:
  tools:
    - name: api
      config_folder: "api_config/"
```

### `api_config/<service_name>.json`

Each file in the folder becomes one service. The filename (without `.json`) is the service name the agent uses.

```json
{
  "serviceName": "My API",
  "serviceDescription": "Brief description of what this service does (injected into the agent prompt).",
  "headers": {
    "Authorization": "Bearer ${MY_API_TOKEN}"
  },
  "endpoints": [
    {
      "name": "get_user",
      "url": "https://api.example.com/users/{user_id}",
      "method": "GET",
      "description": "Get a user by ID",
      "url_parameters": "user_id: string - The ID of the user to retrieve",
      "query_params": "include_posts: boolean - Whether to include user posts"
    },
    {
      "name": "create_post",
      "url": "https://api.example.com/posts",
      "method": "POST",
      "description": "Create a new post",
      "payload": "title: string - Post title. Required.\ncontent: string - Post content. Required."
    }
  ]
}
```

### Service-level fields

| Field | Required | Description |
|---|---|---|
| `serviceName` | no | Display name for the service in prompts (e.g. "Cloudinary"). Falls back to filename if omitted. |
| `serviceDescription` | no | Human-readable description of the service, injected into the tool prompt. |

### Endpoint fields

| Field | Required | Description |
|---|---|---|
| `name` | yes | Identifier the agent uses to call this endpoint |
| `url` | yes | URL template — supports `{param}` and `${ENV_VAR}` placeholders |
| `method` | yes | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `description` | yes | What the endpoint does (shown in `show_apis`) |
| `url_parameters` | no | Free-text description of URL path parameters (shown in `show_api`) |
| `query_params` | no | Free-text description of query string parameters (shown in `show_api`) |
| `payload` | no | Free-text description of body parameters (shown in `show_api`) |
| `content_type` | no | `"form"` for `application/x-www-form-urlencoded`; omit for JSON (default) |

### Authentication options

**Bearer token** (via header):
```json
{
  "headers": {
    "Authorization": "Bearer ${API_TOKEN}"
  }
}
```

**Basic Auth** (auto base64-encoded):
```json
{
  "basic_auth": "${API_KEY}:${API_SECRET}"
}
```

Both values are resolved from environment variables at call time.

## Agent workflow

The tool exposes three actions:

### 1. `show_apis` — list endpoints

```json
{
  "action": "show_apis",
  "service": "my_api"
}
```

Response:
```
Service: my_api
Endpoints:
  - get_user: Get a user by ID
  - create_post: Create a new post
```

### 2. `show_api` — endpoint details

```json
{
  "action": "show_api",
  "service": "my_api",
  "endpoint": "get_user"
}
```

Response:
```
Service: my_api
Endpoint: get_user
Method: GET https://api.example.com/users/{user_id}
Description: Get a user by ID
URL Parameters:
  user_id: string - The ID of the user to retrieve
Query Parameters:
  include_posts: boolean - Whether to include user posts
```

### 3. Call an endpoint

```json
{
  "action": "get_user",
  "service": "my_api",
  "url_params": { "user_id": "123" },
  "query_params": { "include_posts": true }
}
```

```json
{
  "action": "create_post",
  "service": "my_api",
  "body": { "title": "Hello", "content": "World" }
}
```

## Resolver functions

Body arguments whose key matches `<resolver>_$<param_name>` are transformed before the HTTP request is sent. The resolved value replaces the entry under `<param_name>`.

### Convention

In the service JSON, annotate a payload parameter with the resolver prefix:

```
"payload": "resolve_to_base64_$file: local file path to upload. Required."
```

The agent passes:

```json
{
  "action": "upload_image",
  "service": "cloudinary",
  "body": {
    "resolve_to_base64_$file": "/home/user/photo.jpg",
    "upload_preset": "my_preset"
  }
}
```

The tool splits `resolve_to_base64_$file` → resolver `resolve_to_base64`, param `file`, reads the file, and sends `file=data:image/jpeg;base64,...` in the request body.

### Built-in resolvers

| Resolver | Input | Output |
|---|---|---|
| `resolve_to_base64` | Local file path (relative paths are resolved from `working_dir`) | `data:<mime>;base64,<encoded>` data URI |

Supported MIME types for `resolve_to_base64`: `.jpg`/`.jpeg` → `image/jpeg`, `.png` → `image/png`, `.gif` → `image/gif`, `.webp` → `image/webp`, other → `application/octet-stream`.

### Adding a resolver

Add one entry to `resolverRegistry` in `src/tools/api/resolvers.go`:

```go
var resolverRegistry = map[string]ResolverFunc{
    "resolve_to_base64": resolveToBase64,
    "my_resolver":       myResolverFunc,  // add here
}
```

## Response format

```
API Response
Service: my_api
Endpoint: get_user
Method: GET
URL: https://api.example.com/users/123?include_posts=true
Status: 200

Response Headers:
  Content-Type: application/json

Response Body:
{"id": "123", "name": "Jane Doe", ...}
```

## Adding a new service

1. Create `api_config/<service_name>.json` with headers and endpoints
2. Add any required env vars to `.env`
3. Restart the agent

No code changes needed.

## Programmatic usage

```go
import "github.com/thinktwiceco/agent-forge/src/tools/api"

services := map[string]api.ServiceConfig{
    "my_api": {
        Headers: map[string]string{
            "Authorization": "Bearer ${MY_TOKEN}",
        },
        Endpoints: []api.Endpoint{
            {
                Name:        "get_user",
                URL:         "https://api.example.com/users/{user_id}",
                Method:      "GET",
                Description: "Get a user by ID",
                URLParameters: "user_id: string - The user ID",
            },
        },
    },
}

repositoryDir := ""   // path to api_config for install_api_config (workingDir/api_config when using builder); empty disables
workingDir := "/path/to/agent/working/dir"  // base path for resolving relative file paths in resolvers
tool := api.NewApiTool("api", services, repositoryDir, workingDir)
```

## Architecture

| File | Responsibility |
|---|---|
| `types.go` | `Endpoint`, `ServiceConfig`, `Api`, `apiResponse` structs |
| `validate.go` | Internal helpers: `findService`, `findEndpoint`, `validateService` |
| `handler.go` | Action dispatch (`show_apis`, `show_api`, call); env var resolution; body serialization |
| `resolvers.go` | `ResolverFunc` type, `resolverRegistry`, `resolveBodyArgs`, `resolveToBase64` |
| `url_builder.go` | `{param}` substitution and query string building |
| `request.go` | HTTP client execution; `Content-Type` selection based on `content_type` field |
| `tool.go` | Constructor `NewApiTool`; parameter and description generation |
