# API Tool

A generic, reusable tool for making HTTP API calls with configurable endpoints, declarative headers, and parameter validation.

## Features

- **Dynamic Endpoint Discovery**: Agent automatically sees all available endpoints with their documentation
- **URL Parameter Substitution**: Support for path parameters like `/users/{user_id}`
- **Query Parameters**: Easy query string building (e.g., `?limit=10&offset=0`)
- **Request Body Support**: Send JSON payloads with POST/PUT/PATCH requests
- **Custom Validation**: Per-endpoint validation functions to ensure parameter safety
- **Declarative Headers**: Inject auth tokens and custom headers via config; values support `${ENV_VAR}` expansion
- **Type Safety**: Built-in parameter validation and type checking

## Usage

### Basic Example

```go
package main

import (
    "github.com/thinktwiceco/agent-forge/src/tools/api"
)

func main() {
    endpoints := []api.Endpoint{
        {
            Name:        "get_user",
            URL:         "https://api.example.com/users/{user_id}",
            Method:      "GET",
            Description: "Get user by ID",
            URLParameters: `- user_id: string - The ID of the user to retrieve`,
            QueryParams: `- include_posts: boolean - Whether to include user posts
- limit: int - Maximum number of posts to include`,
        },
        {
            Name:        "create_post",
            URL:         "https://api.example.com/users/{user_id}/posts",
            Method:      "POST",
            Description: "Create a new post for a user",
            URLParameters: `- user_id: string - The ID of the user`,
            Payload: `- title: string - The post title
- content: string - The post content
- tags: array[string] - Optional tags`,
        },
    }

    headers := map[string]string{
        "Authorization": "Bearer YOUR_API_TOKEN",
    }

    tool := api.NewApiTool("my_api", endpoints, headers)
}
```

### YAML Configuration

```yaml
agent:
  name: "API Agent"
  tools:
    - name: "my_api"
      headers:
        - "Authorization: Bearer ${API_TOKEN}"
        - "X-API-Version: v1"
      endpoints:
        - name: "get_user"
          url: "https://api.example.com/users/{user_id}"
          method: "GET"
          description: "Get user by ID"
          urlParameters: |
            - user_id: string - The ID of the user to retrieve
          queryParams: |
            - include_posts: boolean - Whether to include user posts
            - limit: int - Maximum number of posts to include
          validator: "validate_positive_user_id"
        
        - name: "create_post"
          url: "https://api.example.com/users/{user_id}/posts"
          method: "POST"
          description: "Create a new post for a user"
          urlParameters: |
            - user_id: string - The ID of the user
          payload: |
            - title: string - The post title
            - content: string - The post content
            - tags: array[string] - Optional tags
          validator: "validate_create_post"
```

Header values are expanded at request time using environment variables — `${API_TOKEN}` reads from the process environment, so values like tokens are always fresh.

## Parameter Validation

Register validators to ensure parameters are safe:

```go
package main

import (
    "fmt"
    "github.com/thinktwiceco/agent-forge/src/tools/api"
)

func init() {
    // Validate that user_id is a positive integer
    api.RegisterValidator("validate_positive_user_id", 
        api.ValidatePositiveIntParam("user_id"))
    
    // Custom validation for post creation
    api.RegisterValidator("validate_create_post", 
        func(params api.EndpointValidationParams) error {
            // Validate body is not empty
            if params.Body == "" {
                return fmt.Errorf("post body cannot be empty")
            }
            
            // Validate user_id is positive
            if userID, ok := params.URLParams["user_id"].(float64); ok {
                if userID <= 0 {
                    return fmt.Errorf("user_id must be positive")
                }
            }
            
            // Validate body size
            if len(params.Body) > 10000 {
                return fmt.Errorf("post body exceeds 10KB limit")
            }
            
            return nil
        })
}
```

## Built-in Validators

The tool provides several built-in validator factories:

### ValidatePositiveIntParam

Ensures specified parameters are positive integers:

```go
api.RegisterValidator("validate_ids", 
    api.ValidatePositiveIntParam("user_id", "post_id"))
```

### ValidateRequiredParams

Ensures specified parameters are present and non-empty:

```go
api.RegisterValidator("validate_required", 
    api.ValidateRequiredParams("api_key", "user_id"))
```

### ValidateBodyMaxSize

Ensures request body doesn't exceed a size limit:

```go
api.RegisterValidator("validate_body_size", 
    api.ValidateBodyMaxSize(50000)) // 50KB limit
```

## Agent Usage

Once configured, the agent can call endpoints like this:

```json
{
  "endpoint": "get_user",
  "url_params": {
    "user_id": "123"
  },
  "query_params": {
    "include_posts": true,
    "limit": 10
  }
}
```

```json
{
  "endpoint": "create_post",
  "url_params": {
    "user_id": "123"
  },
  "body": "{\"title\": \"My Post\", \"content\": \"Hello world!\", \"tags\": [\"test\"]}"
}
```

## Response Format

The tool returns formatted responses:

```
API Response
Endpoint: get_user
Method: GET
URL: https://api.example.com/users/123?include_posts=true&limit=10
Status: 200

Response Headers:
  Content-Type: application/json
  X-RateLimit-Remaining: 999

Response Body:
{"id": "123", "name": "John Doe", "email": "john@example.com", ...}
```

## Error Handling

The tool provides clear error messages:

- **Parameter validation failed**: Custom validation rules were not satisfied
- **Missing required URL parameters**: Not all URL placeholders were filled
- **HTTP 4xx/5xx**: API returned an error status code

## Security Best Practices

1. **Use environment variables for secrets**: Reference tokens as `${MY_TOKEN}` in header values, never hardcode them
2. **Register Validators**: Always validate user IDs, size limits, and required fields
3. **Limit Endpoints**: Only expose endpoints that the agent needs
4. **Monitor API Calls**: Log all API calls for audit purposes
5. **Use HTTPS**: Always use secure connections for API calls

## Advanced Example

Complete example with multiple endpoints, validation, and authentication:

```yaml
agent:
  name: "GitHub Agent"
  model: "gpt-4"
  tools:
    - name: "github_api"
      headers:
        - "Authorization: Bearer ${GITHUB_TOKEN}"
        - "Accept: application/vnd.github.v3+json"
      endpoints:
        - name: "list_issues"
          url: "https://api.github.com/repos/{owner}/{repo}/issues"
          method: "GET"
          description: "List issues for a repository"
          urlParameters: |
            - owner: string - Repository owner
            - repo: string - Repository name
          queryParams: |
            - state: string - Issue state (open, closed, all)
            - per_page: int - Results per page (max 100)
          validator: "validate_repo_params"
        
        - name: "get_issue"
          url: "https://api.github.com/repos/{owner}/{repo}/issues/{issue_number}"
          method: "GET"
          description: "Get a specific issue"
          urlParameters: |
            - owner: string - Repository owner
            - repo: string - Repository name
            - issue_number: int - Issue number
          validator: "validate_issue_number"
```

Set `GITHUB_TOKEN` in your environment or `.env` file — it is expanded at request time.

## Architecture

The API tool consists of several components:

- **types.go**: Core data structures (Endpoint, Api, apiResponse)
- **validate.go**: Validation functions and registry
- **url_builder.go**: URL parameter substitution and query string building
- **request.go**: HTTP client and request execution
- **handler.go**: Main tool handler; resolves `${ENV_VAR}` in headers at request time
- **tool.go**: Tool constructor and description generation

## Integration

The tool integrates with the agent builder system and can be configured via:

1. **Direct instantiation**: `api.NewApiTool()`
2. **YAML configuration**: Via agent config files
3. **Builder API**: Using `AgentBuilder.AddTools()`
