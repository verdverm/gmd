my (human) ideas & questions

- create/output schema for collection on init/add
- can we update/migrate collection schemas?
- wiki init should also init collection (if it doesn't)
- CRUD commands for collection dirs
- web crawl
- anthropic endpoint support
- auth/apikeys per model
    - opencode-go support
    - gcloud auth support
- code fetching & processing (like web... + wiki)
    - prefer clone, need a "raw" place for it
- typesense lifecycle commands (backup,restore) perhaps via the collection cmd
- verbose mode with wiki so we know what query pipeline is used (want to see more advanced takes)

---

## Review

in-code: `XXX-AGENT ...` comments


- ./pkg/runtime ought to have more on it, like LLMs?
- ./pkg/web vs ./pkg/exa being separate
- ./pkg/output is only for search it seems...

- count tokens with endpoint, not bad approximation
- remove markdown/other that should be embeded

## Fixed

- inconsistent ./cmd/gmd/... organization / file naming
- ./pkg/agents
    - rename to agentsmd
    - options from file names, not hardcoded