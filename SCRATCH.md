# SCRATCH

## Top Priority

update/enhance docs for changes since 2ea2ab4

## Random

- filters/facets for coll/wiki search
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