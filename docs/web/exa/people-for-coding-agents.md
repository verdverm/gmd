> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# People Search Reference

> Self-contained reference for coding agents using Exa People Search

## Overview

**Endpoint:** `POST https://api.exa.ai/search` with `"category": "people"`

**What it searches:** 1B+ public professional profiles aggregated from LinkedIn, company pages, and other sources. Index refreshed weekly. Semantic search over structured attributes (role, skill, company, location, seniority). Natural language queries return relevance-ranked people results via API.

For creating lists or enriching over many people at scale, use [Websets](/websets/api-guide).

## Minimal Working Example

```bash theme={null}
curl -X POST "https://api.exa.ai/search" \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_KEY" \
  -d '{"query": "senior ML engineers at fintech companies", "category": "people", "contents": {"highlights": true}}'
```

```python theme={null}
from exa_py import Exa
exa = Exa(api_key="YOUR_API_KEY")
result = exa.search("senior ML engineers at fintech companies", category="people", contents={"highlights": True})
```

```javascript theme={null}
import Exa from "exa-js";
const exa = new Exa("YOUR_API_KEY");
const result = await exa.search("senior ML engineers at fintech companies", { category: "people", contents: { highlights: true } });
```

## Parameter Restrictions

The `people` category does **not** support the following parameters. Using them returns a **400 error**:

| Unsupported Parameter | Workaround                                               |
| --------------------- | -------------------------------------------------------- |
| `startPublishedDate`  | Not available. People profiles don't have publish dates. |
| `endPublishedDate`    | Not available.                                           |
| `excludeDomains`      | Not available.                                           |
| `includeDomains`      | Not available.                                           |

## Supported Parameters

| Parameter    | Type    | Notes                                                                 |
| ------------ | ------- | --------------------------------------------------------------------- |
| `query`      | string  | Natural language. Supports role, skill, company, location, seniority. |
| `category`   | string  | Must be `"people"`.                                                   |
| `type`       | string  | `"auto"` recommended. `"deep"` and `"deep-reasoning"` also work.      |
| `numResults` | integer | 1–100. Default 10.                                                    |
| `contents`   | object  | `text`, `highlights`, `summary`, all nested under `contents`.         |

## Structured Entity Metadata

People Search returns structured person metadata in `entities` for result rows that resolve to a person. Each person entity has `type: "person"`, a stable `id`, a schema `version`, and a `properties` object with person profile fields.

```json theme={null}
{
  "title": "Jane Doe - VP Engineering",
  "url": "https://www.linkedin.com/in/janedoe",
  "entities": [
    {
      "id": "person_...",
      "type": "person",
      "version": 1,
      "properties": {
        "name": "Jane Doe",
        "firstName": "Jane",
        "lastName": "Doe",
        "location": "San Francisco, California, United States",
        "workHistory": [
          {
            "title": "VP Engineering",
            "location": "San Francisco, California, United States",
            "dates": { "from": "2022-01-01", "to": null },
            "company": { "id": "company_...", "name": "Example AI" }
          }
        ],
        "educationHistory": [
          {
            "degree": "BS Computer Science",
            "dates": { "from": "2010", "to": "2014" },
            "institution": { "id": null, "name": "Stanford University" }
          }
        ]
      }
    }
  ]
}
```

| Field                                       | Type           | Notes                                                                                                                           |
| ------------------------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `entities[].id`                             | string         | Stable person entity identifier.                                                                                                |
| `entities[].type`                           | string         | `"person"`.                                                                                                                     |
| `entities[].version`                        | integer        | Entity schema version.                                                                                                          |
| `properties.name`                           | string \| null | Full name.                                                                                                                      |
| `properties.firstName`                      | string \| null | First name.                                                                                                                     |
| `properties.lastName`                       | string \| null | Last name.                                                                                                                      |
| `properties.location`                       | string \| null | Person location.                                                                                                                |
| `properties.workHistory`                    | array          | Known roles. Each item has `title`, `location`, `dates`, and `company`.                                                         |
| `properties.workHistory[].dates`            | object \| null | `{ "from": string \| null, "to": string \| null }`. `to: null` usually means current when the source represents an active role. |
| `properties.workHistory[].company`          | object \| null | Referenced company: `{ "id": string \| null, "name": string \| null }`.                                                         |
| `properties.educationHistory`               | array          | Known education entries. Each item has `degree`, `dates`, and `institution`.                                                    |
| `properties.educationHistory[].institution` | object \| null | Referenced institution: `{ "id": string \| null, "name": string \| null }`.                                                     |

Top-level property keys are present on person entities. Treat individual values and nested fields defensively because profile sources can vary in the information they include.

## Query Patterns

**By role and company:**

```
"product managers at Microsoft"
"enterprise sales reps from Salesforce in EMEA"
```

**By skill set:**

```
"machine learning engineer with PyTorch experience"
"full-stack developer React and Node.js"
```

**By seniority and location:**

```
"VP Engineering AI infrastructure San Francisco"
"CTO at fintech startups in New York"
```

**Composite:**

```
"senior data scientists at Series B healthcare companies in Boston"
"DevOps engineers with Kubernetes experience at Fortune 500 companies"
```

## Common Mistakes

| Wrong                                                        | Correct                                                                                                                         |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| `excludeDomains: [...]` with `category: "people"`            | Remove `excludeDomains`. Not supported for people. Returns 400.                                                                 |
| `startPublishedDate: "2025-01-01"` with `category: "people"` | Remove date filters. Not supported for people. Returns 400.                                                                     |
| Missing `category: "people"`                                 | Without `category`, the search runs against the general web index, not the people index. Always include `"category": "people"`. |

## Patterns and Gotchas

* **Always set `category: "people"`.** Without it, you search the general web index and won't get structured people results.
* **Use `highlights` for agent workflows.** People profiles are dense. Highlights extract the most relevant career details without flooding your context window.
* **Use `entities` for typed metadata.** Read names, locations, work history, and education history from `results[].entities[].properties`; use `text` or `highlights` for supporting snippets.
* **Natural language is the only filter.** Since date filters, text filters, and domain filters aren't supported, encode all constraints in your query string.
* **Python SDK uses snake\_case.** `numResults` → `num_results`, `maxCharacters` → `max_characters`.
* **Combine with deep search for custom enrichment.** Use `type: "deep"` with `outputSchema` when you need fields outside the built-in person entity schema.
