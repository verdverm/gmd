> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Code Search Reference

> Self-contained reference for coding agents using Exa Code Search

## Overview

**Endpoint:** `POST https://api.exa.ai/search`. Code search is integrated into the main search endpoint. No category parameter needed.

**What it searches:** Billions of GitHub repositories, documentation pages, Stack Overflow posts, and developer blogs. Semantic search matches natural language queries to real, working code examples ranked by relevance. Reduces hallucinated imports and outdated syntax.

## Minimal Working Example

```bash theme={null}
curl -X POST "https://api.exa.ai/search" \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_KEY" \
  -d '{"query": "how to use Vercel AI SDK streaming", "type": "fast", "contents": {"highlights": true}}'
```

```python theme={null}
from exa_py import Exa
exa = Exa(api_key="YOUR_API_KEY")
result = exa.search("how to use Vercel AI SDK streaming", type="fast", contents={"highlights": True})
```

```javascript theme={null}
import Exa from "exa-js";
const exa = new Exa("YOUR_API_KEY");
const result = await exa.search("how to use Vercel AI SDK streaming", { type: "fast", contents: { highlights: true } });
```

## Supported Parameters

Code search is integrated into the main search endpoint. All standard search parameters are supported:

| Parameter            | Type      | Notes                                                                                              |
| -------------------- | --------- | -------------------------------------------------------------------------------------------------- |
| `query`              | string    | Natural language describing the code you need. Be specific about language, framework, and version. |
| `type`               | string    | `"fast"` recommended for code search. All search types supported.                                  |
| `numResults`         | integer   | 1–100. Default 10.                                                                                 |
| `includeDomains`     | string\[] | Restrict to specific sources (e.g. `["github.com", "stackoverflow.com"]`).                         |
| `excludeDomains`     | string\[] | Exclude specific sources.                                                                          |
| `startPublishedDate` | string    | ISO 8601. Filter for recent code examples.                                                         |
| `endPublishedDate`   | string    | ISO 8601.                                                                                          |
| `contents`           | object    | `text`, `highlights`, `summary`, all nested under `contents`.                                      |

## Query Patterns

**Library usage:**

```
"how to use Exa search in python with livecrawl"
"pandas dataframe filtering and groupby operations"
```

**API syntax:**

```
"correct syntax for vercel ai sdk to call gpt-5"
"Next.js 14 app router with TypeScript configuration"
```

**Development setup:**

```
"how to set up a reproducible Nix Rust development environment"
"Docker Compose for PostgreSQL and Redis"
```

**Framework-specific:**

```
"React Server Components data fetching patterns"
"FastAPI dependency injection with SQLAlchemy"
```

## Common Mistakes

| Wrong                                  | Correct                                                                                                                                |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Vague queries like `"python code"`     | Be specific: `"python asyncio gather with error handling"`. The more specific the query, the better the code results.                  |
| Using `text: true` for code extraction | Use `highlights` for targeted code snippets, or `text` with `maxCharacters` to cap output size. Raw `text: true` returns entire pages. |

## Patterns and Gotchas

* **Code search is integrated into the main search endpoint.** No category parameter needed. Just use descriptive code-related queries with `type: "fast"`.
* **Be specific about language and framework.** "how to stream responses with Vercel AI SDK in Next.js" returns much better results than "streaming API".
* **Use `highlights` to extract code snippets.** Highlights pull the most relevant code blocks from pages, avoiding boilerplate and navigation text.
* **`includeDomains` is useful for source quality.** Restrict to `["github.com"]` for raw code, `["stackoverflow.com"]` for Q\&A, or official docs domains.
* **Date filters work for code.** Use `startPublishedDate` to get recent examples, useful for fast-moving frameworks.
* **Python SDK uses snake\_case.** `numResults` → `num_results`, `maxCharacters` → `max_characters`.
* **Use `text` with `maxCharacters` for full context.** When you need the complete code file or tutorial, request `text` with a character cap rather than just highlights.
