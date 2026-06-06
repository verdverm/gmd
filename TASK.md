## multi-llm providers and profiles

Currently, we only have basic support for llms. We want to expand in the following ways:
- support multiple llm providers beyond openai compat, like athropic (which vllm might also provide?)
- support multiple llm provider auth methods: [local,opencode,vertex]
- each of the configurable llm roles we have should be turned into an object, we will have new schemas
- support profiles for configuration generally

## feedback on plan v1

we should have both auth method and provider fields
  - auth is more like [none,apikey,service-account,...]
  - provider is [openai,anthropic,vertex,opencode]
is there some go module that already handles this for us, such that we do not have to create provider packages for each? Can we get this down to mostly config/data processing?

