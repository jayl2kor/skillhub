package skill

import "fmt"

// AgentType represents a supported AI agent tool.
type AgentType struct {
	Name       string // e.g. "claude", "cursor"
	SkillsPath string // relative path from project root, e.g. ".claude/skills"
}

var knownAgents = map[string]AgentType{
	"claude":  {Name: "claude", SkillsPath: ".claude/skills"},
	"cursor":  {Name: "cursor", SkillsPath: ".cursor/skills"},
	"windsurf": {Name: "windsurf", SkillsPath: ".windsurf/skills"},
	"cline":   {Name: "cline", SkillsPath: ".cline/skills"},
	"generic": {Name: "generic", SkillsPath: ".ai/skills"},
}

// LookupAgent returns the agent type definition for the given name.
func LookupAgent(name string) (AgentType, error) {
	a, ok := knownAgents[name]
	if !ok {
		return AgentType{}, fmt.Errorf("unknown agent type %q (supported: claude, cursor, windsurf, cline, generic)", name)
	}
	return a, nil
}

// AllAgentNames returns all supported agent type names.
func AllAgentNames() []string {
	names := make([]string, 0, len(knownAgents))
	for name := range knownAgents {
		names = append(names, name)
	}
	return names
}
