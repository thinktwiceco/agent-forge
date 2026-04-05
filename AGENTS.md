agents | index

R1|read first | R2|reference | R3|how-to | R4|constraints

rules

R5|machine-first|thinking,language,always
R6|update refs|on change,touches agents.md
R7|propose refactor|module too big|>400 LOC per file
R8|maintain patterns|existing,always
R9|no XML tags|system prompts,tool descriptions|use [brackets] not <tags>
R10|no Legacy implementations. Ask to the user before implementing any legacy/retrocompatible code

docs/agents

R1|framework,purpose,principles,packages|docs/agents/overview.md
R1|deps,structure,imports|docs/agents/architecture.md
R2|builder,config,env|docs/agents/configuration.md
R2|brain plugin memory,dreaming|docs/agents/configuration.md#brain-plugin-yaml,src/plugins/README.md#brain-plugin,memorySpec.md
R2|commands,build,test,lint|docs/agents/quickref.md
R2|error handling,interfaces,conventions|docs/agents/patterns.md
R2|formatting,naming,style|docs/agents/code-style.md
R3|add tool,package,impl|docs/agents/how-to-tools.md
R3|plugin,hook,tool provider|docs/agents/how-to-plugins.md
R4|boundaries,permissions,ask-before|docs/agents/safety.md
R4|debug,common issues|docs/agents/troubleshooting.md
R2|unit,mocks,coverage|docs/agents/testing.md
