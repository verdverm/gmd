# feedback on design

Generally, the spec is the truth, we should not try to bridge with 'gmd wiki'

1. remove difficulty from wiki frontmatter
2. I'm not sure where you are getting "body convention sections", did you make that up from the examples or are there spec entries that make this explicit? Either way, I disagree because there are numerous examples that these make zero sense for. We can have several examples, which depend on the kind or concept (tbd, make a few up)
3. CUE Schema (1) frontmatter should not have a literal "fields" field, it is a simple map (2) we should ensure required fields are required in the schema (3) common fields with well known validation should be marked as optional with a good validation value for CUE


Open questions

1. We do not do backwards compatibility, again... alpha software...
2. There should not be any fixed structure to wiki, if it is there now, it's very wrong
3. No, why are you even asking...?
4. There are NO internal categories... free form, there will be no importing of other wikis...
5. It should be deterministic and enforced, do not let dumb agents try to remember to do this
6. yes, use the summarizer llm 
7. Do not worry about this, the MCP is completely unimplemented and not in scope
8. more information is better, keep it in the heading