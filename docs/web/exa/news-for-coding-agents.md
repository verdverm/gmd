> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# News Search Reference

> Self-contained reference for coding agents using Exa News Search

## Overview

**Endpoint:** `POST https://api.exa.ai/search`. News search is integrated into the main search endpoint. No category parameter needed.

**What it searches:** Real-time index of web news sources including major publications, trade press, and niche outlets. Semantic search returns results ranked by topical relevance.

## Minimal Working Example

```bash theme={null}
curl -X POST "https://api.exa.ai/search" \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_KEY" \
  -d '{"query": "AI regulation updates", "type": "auto", "contents": {"highlights": true}}'
```

```python theme={null}
from exa_py import Exa
exa = Exa(api_key="YOUR_API_KEY")
result = exa.search("AI regulation updates", type="auto", contents={"highlights": True})
```

```javascript theme={null}
import Exa from "exa-js";
const exa = new Exa("YOUR_API_KEY");
const result = await exa.search("AI regulation updates", { type: "auto", contents: { highlights: true } });
```

## Supported Parameters

News search is integrated into the main search endpoint. All standard search parameters are supported:

| Parameter            | Type      | Notes                                                                         |
| -------------------- | --------- | ----------------------------------------------------------------------------- |
| `query`              | string    | Natural language. Topic, company, person, or event.                           |
| `type`               | string    | `"auto"` recommended. All search types supported.                             |
| `numResults`         | integer   | 1–100. Default 10.                                                            |
| `includeDomains`     | string\[] | Restrict to specific publications (e.g. `["reuters.com", "techcrunch.com"]`). |
| `excludeDomains`     | string\[] | Exclude specific sources.                                                     |
| `startPublishedDate` | string    | ISO 8601. Limits to recent articles.                                          |
| `endPublishedDate`   | string    | ISO 8601. Upper bound on publication date.                                    |
| `contents`           | object    | `text`, `highlights`, `summary`, all nested under `contents`.                 |

## Query Patterns

**Industry news:**

```
"AI startup funding announcements"
"semiconductor supply chain disruptions"
```

**Company-specific coverage:**

```
"OpenAI product launches"
"Tesla quarterly earnings results"
```

**Geopolitical events:**

```
"trade policy changes US China"
"climate summit agreements 2026"
```

**Time-bounded monitoring:**

```json theme={null}
{
  "query": "cybersecurity breaches",
  "type": "auto",
  "numResults": 20
}
```

## Common Mistakes

| Wrong                       | Correct                                                                                                                 |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Vague queries like `"news"` | Be specific: `"AI regulation updates in the European Union"`. The more specific the query, the better the news results. |

## Patterns and Gotchas

* **News search is integrated into the main search endpoint.** No category parameter needed. Just use descriptive news-related queries with `type: "auto"`.
* **`includeDomains` controls source quality.** For trusted sources, restrict to `["reuters.com", "bbc.com", "nytimes.com"]`. For trade press, use `["techcrunch.com", "theverge.com"]`.
* **Use `highlights` for agent workflows.** News articles are verbose. Highlights extract the key facts and quotes.
* **Python SDK uses snake\_case.** `numResults` → `num_results`, `maxCharacters` → `max_characters`.
* **News works well with deep search.** Use `type: "deep"` with `outputSchema` to extract structured event summaries, sentiment, or entity mentions from news results.
