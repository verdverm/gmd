package gmd

Config: {
	collections: docs: {
		path:    "."
		patterns: ["**/*.md"]
		ignore:  [
			"qmd/**",
			"tmp/**",
			"pkg/agentsmd/content/**",
			"pkg/wiki/skills/**",
		]
		context: "Project documentation"

		// Optional: define frontmatter fields to index for faceted search/filtering.
		// Fields must match YAML frontmatter keys in your markdown files.
		// Supported types: string, string[], int32, float, bool
		// fields: {
		// 	tags:  { type: "string[]", facet: true }
		// 	author: { type: "string", facet: true }
		// 	rating: { type: "float", sort: true }
		// }
	}
}
