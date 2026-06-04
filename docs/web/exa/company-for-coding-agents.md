> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Company Search Reference

> Self-contained reference for coding agents using Exa Company Search

## Overview

**Endpoint:** `POST https://api.exa.ai/search` with `"category": "company"`

**What it searches:** 50M+ company pages including LinkedIn company profiles, official websites, and Crunchbase-style data. Semantic search over industry, funding stage, headcount, geography, and technology attributes. Natural language queries return relevance-ranked company results.

For creating lists or enriching over many companies at scale, use [Websets](/websets/api-guide).

## Minimal Working Example

```bash theme={null}
curl -X POST "https://api.exa.ai/search" \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_KEY" \
  -d '{"query": "fintech companies in Switzerland", "category": "company", "contents": {"highlights": true}}'
```

```python theme={null}
from exa_py import Exa
exa = Exa(api_key="YOUR_API_KEY")
result = exa.search("fintech companies in Switzerland", category="company", contents={"highlights": True})
```

```javascript theme={null}
import Exa from "exa-js";
const exa = new Exa("YOUR_API_KEY");
const result = await exa.search("fintech companies in Switzerland", { category: "company", contents: { highlights: true } });
```

## Parameter Restrictions

The `company` category does **not** support the following parameters. Using them returns a **400 error**:

| Unsupported Parameter | Workaround                                                       |
| --------------------- | ---------------------------------------------------------------- |
| `startPublishedDate`  | Not available. Use natural language (e.g. "founded after 2020"). |
| `endPublishedDate`    | Not available.                                                   |
| `excludeDomains`      | Not available.                                                   |

## Supported Parameters

| Parameter    | Type    | Notes                                                                                       |
| ------------ | ------- | ------------------------------------------------------------------------------------------- |
| `query`      | string  | Natural language. Supports industry, geography, funding, headcount, technology, similarity. |
| `category`   | string  | Must be `"company"`.                                                                        |
| `type`       | string  | `"auto"` recommended. `"deep"` and `"deep-reasoning"` also work.                            |
| `numResults` | integer | 1–100. Default 10.                                                                          |
| `contents`   | object  | `text`, `highlights`, `summary`, all nested under `contents`.                               |

## Structured Entity Metadata

Company Search returns structured company metadata in `entities` for result rows that resolve to a company. Each company entity has `type: "company"`, a stable `id`, a schema `version`, and a `properties` object with company profile fields.

```json theme={null}
{
  "title": "Example AI",
  "url": "https://www.example.ai",
  "entities": [
    {
      "id": "company_...",
      "type": "company",
      "version": 1,
      "properties": {
        "name": "Example AI",
        "foundedYear": 2021,
        "description": "AI infrastructure company for enterprise search.",
        "workforce": { "total": 120 },
        "headquarters": {
          "address": "123 Market Street",
          "city": "San Francisco",
          "postalCode": "94105",
          "country": "United States"
        },
        "financials": {
          "revenueAnnual": null,
          "fundingTotal": 42000000,
          "fundingLatestRound": {
            "name": "Series B",
            "date": "2025-03-15",
            "amount": 30000000
          }
        },
        "webTraffic": {
          "visitsMonthly": 250000,
          "countryRank": 12000,
          "avgDurationSeconds": 180,
          "history": [
            { "value": 250000, "dateFrom": "2026-04", "dateTo": "2026-04" }
          ]
        }
      }
    }
  ]
}
```

| Field                                      | Type            | Notes                                                                               |
| ------------------------------------------ | --------------- | ----------------------------------------------------------------------------------- |
| `entities[].id`                            | string          | Stable company entity identifier.                                                   |
| `entities[].type`                          | string          | `"company"`.                                                                        |
| `entities[].version`                       | integer         | Entity schema version.                                                              |
| `properties.name`                          | string \| null  | Company name.                                                                       |
| `properties.foundedYear`                   | integer \| null | Year the company was founded.                                                       |
| `properties.description`                   | string \| null  | Short company description.                                                          |
| `properties.workforce`                     | object \| null  | Workforce details. Currently includes `total`, the estimated employee count.        |
| `properties.headquarters`                  | object \| null  | `address`, `city`, `postalCode`, and `country`.                                     |
| `properties.financials`                    | object \| null  | `revenueAnnual`, `fundingTotal`, and `fundingLatestRound`, all in USD when present. |
| `properties.financials.fundingLatestRound` | object \| null  | `name`, `date`, and `amount` for the most recent funding round.                     |
| `properties.webTraffic`                    | object \| null  | `visitsMonthly`, `countryRank`, `avgDurationSeconds`, and monthly `history`.        |
| `properties.webTraffic.history[]`          | object          | Historical monthly visits with `value`, `dateFrom`, and `dateTo`.                   |

Top-level property keys are present on company entities. Treat individual values and nested objects defensively because sources can vary in the information they include.

## Query Patterns

**Named lookup:**

```
"Sakana AI company"
"Tell me about exa.ai"
```

**Attribute filtering:**

```
"fintech companies in Switzerland"
"Japanese AI companies founded in 2023"
```

**Funding queries:**

```
"agtech companies in the US that have raised series A"
"startups that raised 30M to 80M"
```

**Composite queries:**

```
"Israeli security companies founded after 2015"
"German enterprise SaaS companies with more than 500 employees"
```

**Semantic / similarity:**

```
"Companies like Bell Labs"
"Companies working on making space travel cheaper"
"competitors of Notion"
```

## Common Mistakes

| Wrong                                                         | Correct                                                                                                    |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `excludeDomains: [...]` with `category: "company"`            | Remove `excludeDomains`. Not supported for company. Returns 400.                                           |
| `startPublishedDate: "2023-01-01"` with `category: "company"` | Remove date filters. Use natural language like "founded after 2023" in query.                              |
| Missing `category: "company"`                                 | Without `category`, the search runs against the general web index. Always include `"category": "company"`. |

## Patterns and Gotchas

* **Always set `category: "company"`.** Without it, you search the general web index and won't get company-specific results.
* **Natural language handles what filters can't.** Since date/text/exclude filters aren't supported, put all constraints in your query: "Series A fintech companies in Europe with 50-200 employees founded after 2020".
* **Use `highlights` for agent workflows.** Company pages are long. Highlights extract key details (industry, funding, headcount) efficiently.
* **Use `entities` for typed metadata.** Read founded year, workforce, headquarters, financials, and web traffic from `results[].entities[].properties`; use `text` or `highlights` for supporting snippets.
* **Similarity queries work well.** "Companies like X" and "competitors of X" leverage semantic understanding of the company index.
* **Python SDK uses snake\_case.** `numResults` → `num_results`, `maxCharacters` → `max_characters`.
* **Combine with deep search for custom enrichment.** Use `type: "deep"` with `outputSchema` when you need fields outside the built-in company entity schema.
