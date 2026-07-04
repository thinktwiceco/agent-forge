package core

// ConfigWriter mutates agent configuration on disk. Localforge provides the
// implementation; library-only runs leave Writer nil and config tool mutations fail gracefully.
type ConfigWriter interface {
	AddTool(name string, params map[string]any) error
	RemoveTool(name string) error
	AddPlugin(name string) error
	RemovePlugin(name string) error
	SetHeartbeat(every string) error
	SetDream(status string, dreamTime string) error
}
