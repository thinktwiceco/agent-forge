# API Tool Quick Start Guide

This guide provides a quick introduction to using the API tool for making HTTP requests from your agents.

## Basic Concept

The API tool allows agents to interact with RESTful APIs by:
1. Defining endpoints with their parameters
2. Letting the agent discover available endpoints
3. Making authenticated API calls automatically
4. Validating parameters for safety

## 5-Minute Setup

### 1. Define Your Endpoints

```go
endpoints := []api.Endpoint{
    {
        Name:        "get_user",
        URL:         "https://api.example.com/users/{user_id}",
        Method:      "GET",
        Description: "Get user by ID",
        URLParameters: `- user_id: string - The user ID`,
    },
}
```

### 2. Create Authentication Hook (Optional)

```go
authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["Authorization"] = "Bearer YOUR_TOKEN"
    return headers, nil
}
```

### 3. Create and Add Tool

```go
apiTool := api.NewApiTool("my_api", endpoints, authHook)
agent.AddTools([]llms.Tool{apiTool})
```

### 4. Use It!

Ask your agent: *"Get information about user 123"*

The agent will automatically:
- Select the `get_user` endpoint
- Extract `user_id: "123"` from your request
- Make the API call: `GET https://api.example.com/users/123`
- Return the formatted response

## Common Patterns

### Public API (No Auth)

```go
endpoints := []api.Endpoint{
    {
        Name:        "get_pokemon",
        URL:         "https://pokeapi.co/api/v2/pokemon/{name}",
        Method:      "GET",
        Description: "Get Pokemon information",
        URLParameters: `- name: string - Pokemon name or ID`,
    },
}

apiTool := api.NewApiTool("pokemon_api", endpoints, nil)
```

### Bearer Token Auth

```go
authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["Authorization"] = "Bearer " + os.Getenv("API_TOKEN")
    return headers, nil
}
```

### API Key in Header

```go
authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
    headers["X-API-Key"] = os.Getenv("API_KEY")
    return headers, nil
}
```

### POST with JSON Body

```go
{
    Name:        "create_post",
    URL:         "https://api.example.com/posts",
    Method:      "POST",
    Description: "Create a new post",
    Payload: `- title: string - Post title
- content: string - Post content
- tags: array[string] - Optional tags`,
}
```

### Query Parameters

```go
{
    Name:        "search_users",
    URL:         "https://api.example.com/users",
    Method:      "GET",
    Description: "Search users with filters",
    QueryParams: `- name: string - Filter by name
- age_min: int - Minimum age
- age_max: int - Maximum age
- limit: int - Results per page (default: 20)`,
}
```

## Parameter Validation

Add validation to ensure parameters are safe:

```go
// Register validators (do this once at startup)
func init() {
    api.RegisterValidator("positive_id", 
        api.ValidatePositiveIntParam("user_id", "post_id"))
    
    api.RegisterValidator("required_fields",
        api.ValidateRequiredParams("api_key", "user_id"))
}

// Use in endpoints
endpoints := []api.Endpoint{
    {
        Name:     "get_user",
        URL:      "https://api.example.com/users/{user_id}",
        Method:   "GET",
        Validate: api.GetValidator("positive_id"),
    },
}
```

## YAML Configuration

For production, use YAML configuration:

```yaml
# config.yaml
agent:
  name: "API Agent"
  tools:
    - name: "api"
      endpoints:
        - name: "get_user"
          url: "https://api.example.com/users/{user_id}"
          method: "GET"
          description: "Get user by ID"
          urlParameters: |
            - user_id: string - The user ID
          validator: "positive_id"
      onApiCallHook: "bearer_auth"
```

Register hooks and validators in your code:

```go
func init() {
    api.RegisterHook("bearer_auth", func(url string, headers map[string]string, body string) (map[string]string, error) {
        headers["Authorization"] = "Bearer " + os.Getenv("API_TOKEN")
        return headers, nil
    })
    
    api.RegisterValidator("positive_id", 
        api.ValidatePositiveIntParam("user_id"))
}
```

## Complete Example

```go
package main

import (
    "os"
    "github.com/thinktwiceco/agent-forge/src/agents"
    "github.com/thinktwiceco/agent-forge/src/tools/api"
)

func main() {
    // Register authentication hook
    api.RegisterHook("github_auth", func(url string, headers map[string]string, body string) (map[string]string, error) {
        headers["Authorization"] = "Bearer " + os.Getenv("GITHUB_TOKEN")
        headers["Accept"] = "application/vnd.github.v3+json"
        return headers, nil
    })
    
    // Define endpoints
    endpoints := []api.Endpoint{
        {
            Name:        "list_repos",
            URL:         "https://api.github.com/users/{username}/repos",
            Method:      "GET",
            Description: "List user repositories",
            URLParameters: `- username: string - GitHub username`,
            QueryParams: `- per_page: int - Results per page (max 100)
- sort: string - Sort by: created, updated, pushed, full_name`,
        },
    }
    
    // Create tool
    apiTool := api.NewApiTool("github_api", endpoints, api.GetHook("github_auth"))
    
    // Add to agent
    agent := agents.NewAgent(&agents.AgentConfig{
        LLMEngine: llm,
        AgentName: "GitHub Assistant",
        Tools:     []llms.Tool{apiTool},
    })
    
    // Use it
    agent.ChatStream("List repositories for user octocat", "")
}
```

## How the Agent Uses It

When you ask: *"List repositories for user octocat"*

The agent:
1. Sees available endpoints in tool description
2. Selects `list_repos` endpoint
3. Extracts parameters: `username: "octocat"`
4. Calls tool with:
   ```json
   {
     "endpoint": "list_repos",
     "url_params": {"username": "octocat"}
   }
   ```
5. Tool makes authenticated request
6. Returns formatted response

## Troubleshooting

### "endpoint not found"
- Check endpoint name matches what you defined
- Ensure tool is properly added to agent

### "missing required URL parameters"
- URL has placeholders like `{user_id}` that weren't filled
- Agent didn't extract parameters correctly - try being more explicit

### "authentication hook failed"
- Check environment variables are set
- Verify hook function returns correct headers

### "parameter validation failed"
- Parameters don't meet validation rules
- Check validator implementation

## Next Steps

- See [API Tool README](../src/tools/api/README.md) for comprehensive documentation
- Check [Pokemon API Example](../examples/README.md) for a working example
- Read [Tools Documentation](TOOLS.md) for other built-in tools

## Best Practices

1. **Security**
   - Never hardcode API tokens in endpoint definitions
   - Use authentication hooks to inject credentials
   - Always validate parameters with validators

2. **Endpoint Design**
   - Use clear, descriptive names
   - Document all parameters thoroughly
   - Group related endpoints in the same tool

3. **Error Handling**
   - Provide helpful error messages
   - Include troubleshooting information
   - Test endpoints before deployment

4. **Performance**
   - Set appropriate timeouts (default: 30s)
   - Limit response body sizes when possible
   - Use validators to prevent expensive operations
