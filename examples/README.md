# Examples

This directory contains example configurations and use cases for the ThinkTwice Agent.

## Pokemon API Example

The Pokemon Agent demonstrates the generic API tool by integrating with the [PokeAPI](https://pokeapi.co/), a free RESTful Pokemon API.

### Configuration

See [`pokemon_agent_config.yaml`](pokemon_agent_config.yaml) for a complete YAML configuration.

### Available Endpoints

1. **get_pokemon** - Get detailed information about a specific Pokemon
   - URL: `https://pokeapi.co/api/v2/pokemon/{name}`
   - Parameters: name (Pokemon name or ID)
   - Example: "Get information about Pikachu"

2. **list_pokemon** - List all Pokemon with pagination
   - URL: `https://pokeapi.co/api/v2/pokemon`
   - Query params: limit, offset
   - Example: "List the first 10 Pokemon"

3. **get_pokemon_species** - Get species information including evolution
   - URL: `https://pokeapi.co/api/v2/pokemon-species/{id}`
   - Parameters: id (species ID or name)
   - Example: "Get evolution chain for Charmander"

4. **get_ability** - Get information about a Pokemon ability
   - URL: `https://pokeapi.co/api/v2/ability/{id}`
   - Parameters: id (ability ID or name)
   - Example: "What does the ability 'Blaze' do?"

5. **get_type** - Get type information and effectiveness
   - URL: `https://pokeapi.co/api/v2/type/{id}`
   - Parameters: id (type ID or name)
   - Example: "What is fire type strong against?"

### Usage Examples

#### Using the CLI (cmd/chat/main.go)

The Pokemon API is already integrated into the chat CLI. Just build and run:

```bash
go build -o chat cmd/chat/main.go
./chat
```

Then ask questions like:
- "Tell me about Pikachu"
- "List the first 20 Pokemon"
- "What type is Charizard?"
- "What abilities does Mewtwo have?"

#### Using YAML Configuration

```bash
# Load the Pokemon agent configuration
agent, err := builder.NewAgentBuilder("Pokemon Agent", "json").
    LoadFromFile("examples/pokemon_agent_config.yaml").
    Build()
```

### Example Queries

**Get a specific Pokemon:**
```
User: Get information about Pikachu

Agent will call:
{
  "endpoint": "get_pokemon",
  "url_params": {
    "name": "pikachu"
  }
}
```

**List Pokemon with pagination:**
```
User: Show me the first 5 Pokemon

Agent will call:
{
  "endpoint": "list_pokemon",
  "query_params": {
    "limit": 5,
    "offset": 0
  }
}
```

**Get Pokemon by ID:**
```
User: What is Pokemon number 25?

Agent will call:
{
  "endpoint": "get_pokemon",
  "url_params": {
    "name": "25"
  }
}
```

### Response Format

The API tool returns formatted responses:

```
API Response
Endpoint: get_pokemon
Method: GET
URL: https://pokeapi.co/api/v2/pokemon/pikachu
Status: 200

Response Headers:
  Content-Type: application/json

Response Body:
{
  "id": 25,
  "name": "pikachu",
  "height": 4,
  "weight": 60,
  "types": [
    {"type": {"name": "electric"}}
  ],
  ...
}
```

## Creating Your Own API Tool

To create your own API integration:

1. **Define Endpoints:**
   ```go
   endpoints := []api.Endpoint{
       {
           Name:        "endpoint_name",
           URL:         "https://api.example.com/resource/{id}",
           Method:      "GET",
           Description: "Description of what this endpoint does",
           URLParameters: "- id: string - Description",
           QueryParams:   "- limit: int - Description",
           Payload:       "- field: string - Description",
       },
   }
   ```

2. **Create Authentication Hook (if needed):**
   ```go
   authHook := func(url string, headers map[string]string, body string) (map[string]string, error) {
       headers["Authorization"] = "Bearer " + os.Getenv("API_TOKEN")
       return headers, nil
   }
   ```

3. **Register Validators (optional):**
   ```go
   api.RegisterValidator("validate_id", 
       api.ValidatePositiveIntParam("id"))
   ```

4. **Create and Add Tool:**
   ```go
   tool := api.NewApiTool("my_api", endpoints, authHook)
   agent.AddTools([]llms.Tool{tool})
   ```

See [`src/tools/api/README.md`](../src/tools/api/README.md) for comprehensive documentation.
