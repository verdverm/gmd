package gmd

// Config is the unified root configuration that all files (global + project-local)
// are unified against. It embeds the schema from types.cue and applies defaults
// from pipeline.cue.
Config: ProjectConfig & {
	pipeline: DefaultPipeline
}
