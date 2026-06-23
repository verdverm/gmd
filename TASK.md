# Rewrite pkg/llm

`pkg/llm` is really poorly written code and interfaces, we need to rewrite it from scratch.

1. It should be built on ADK (see .extern/adk-go), there is an openai client implementation in ~/hof/hof/lib/agent/models
2. We are rethinking the entire package, the Client interface is trash, it's not even a real interface...
3. All consumers need to be re-thought through. There will be significant changes outside of pkg/llm
