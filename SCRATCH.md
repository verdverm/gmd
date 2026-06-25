# SCRATCH


## Go Markdown

- need to writeup some guidelines for Go development and patterns (not necessarily here, but it is impacting here)

## Random

- truncation in fusion, where else?
- (system and other) prompts in Go code
- filters/facets for coll/wiki search
- prompt processing package (what was this?)
- work on agent/research, agentsmd -> craft new command(s) for installing/removing our agents, etc... && 'gdm agent' generally to launch and agent in various modes
    - markdown and skills, run with opencode -c (generally a configurable agent-cli)
    - ability to write these files to global/project with init [--global]
    - writing should support advanced stuff based on harness, make these downloadable and easy install for all of them
- make all the agent prompts and more configurable
- more go lint, vuln checks, readme badges
- produce artifacts and make gmd easy to install
- typesense proxy config / stemming / synonyms
- use CUE loader so we can use proper modules, imports, and unification
- support more llm providers? at least anthropic style endpoints because models...
    - ideally, we call an underlying harness like scion
    - we could even recommend scion
- 'gmd init' should write a global config if global dir not found
- 'gmd find ...' ?
- record some integration (real) results and save them for better regular tests
- always write web search results somewhere
- maybe we end up with an sqlite db after all?
- code index, search, research, explore, AST chunking
- "Codex CLI" in the agent skills writable to disk
- config for persisting 'gmd web search/fetch/crawl' results

---

agents:

- gmd
- personal

## launching agent harnesses from gmd

We want to be able to start a user's preferred harness from the cli, prepopulated with message and other preperation, from various places and ways within the cli
- different ways to do this based on harness, let's just do opencode for now, prepare for expansion
- different places within the cli, infer agent for task based on the ones we provide, which the user may or may not have "installed" yet
- do opencode for now, prepare for expansion to other cli, there should be config for these as well
- we'll want to associate them with agents/providers as well

---


## Old notes

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



web cmd:
- ./pkg/web/exa
- web prompts as embedded files

other:

- ./pkg/runtime ought to have more on it, like LLMs?
- ./pkg/output is only for search it seems...? (not anymore perhaps)
- count tokens with endpoint, not bad approximation
- remove markdown/other that should be embeded

## Fixed

- multi-get should go away, get should be flexible enough itself
- inconsistent ./cmd/gmd/... organization / file naming
- ./pkg/agents
    - rename to agentsmd
    - options from file names, not hardcoded