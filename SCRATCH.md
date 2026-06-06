# SCRATCH

- filters/facets for coll/wiki search
- parallel web search, list in group config, synth/dedup with agent or raw
- move any markdown/config to embedded files
- work on agent/research
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

---

In @README.md there are sections
- Index and search
- Web search...
- LLM Wiki
- Quick start > Configure

1. Only web search has a partial config example
2. Put the Web search... section after LLM Wiki, this should make the three core features consistent order everywhere
3. Add another docker run ... example in Quick start > Start /Typesense/Containers/ for searxng
4. There are a pair of bullets for web/wiki that also need to swap order
