> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# People Search

Exa's people search lets you search over 1B+ professional profiles using natural language. The index is refreshed weekly and combines semantic search with structured attributes such as role, skill, company, location, and seniority.

Read the blog post: [Introducing Exa's People Search Benchmarks](https://exa.ai/blog/people-search-benchmark)

<Tip>
  For agentic list-building and enrichment out of the box, use [Websets →](/websets/api-guide).
</Tip>

## When to Use

If you're building:

* A **recruiting or talent-sourcing platform**
  * Search candidates by role, skill set, location, or employer
  * "senior ML engineers in Seattle with PyTorch experience"
  * "full-stack developers with React and Node.js"
* A **GTM intelligence or sales prospecting tool**
  * Find decision-makers and buying-committee members at target accounts
  * "VP Engineering at Series B fintech companies"
  * "enterprise sales reps from Salesforce in EMEA"
* A **professional services or consulting workflow**
  * Map leadership and org charts at companies you're researching for clients
  * "CTO at fintech startups in New York"
* An **AI SDR or outbound agent**
  * Enrich prospect lists with up-to-date titles, companies, and career context
  * "product managers at Microsoft"

## Basic Usage

<CodeGroup>
  ```bash curl theme={null}
  curl -X POST https://api.exa.ai/search \
    -H "x-api-key: YOUR_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
      "query": "CEO of AI search startups in San Francisco",
      "category": "people",
      "type": "auto",
      "numResults": 10
    }'
  ```

  ```python python theme={null}
  from exa_py import Exa

  exa = Exa(api_key="YOUR_API_KEY")

  results = exa.search(
      "CEO of AI search startups in San Francisco",
      category="people",
      type="auto",
      num_results=10
  )
  ```

  ```javascript javascript theme={null}
  import Exa from "exa-js";

  const exa = new Exa("YOUR_API_KEY");

  const results = await exa.search(
    "CEO of AI search startups in San Francisco",
    {
      category: "people",
      type: "auto",
      numResults: 10,
    }
  );
  ```
</CodeGroup>

## Structured Entity Metadata

People Search returns structured person metadata in `entities` for result rows that resolve to a person. Each person entity has `type: "person"`, a stable `id`, a schema `version`, and a `properties` object with person profile fields.

```json theme={null}
{
  "results": [
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
  ]
}
```

<Info>
  Try it now: [People Search in the API Playground →](https://dashboard.exa.ai/playground/search?q=product%20managers%20at%20microsoft\&c=people\&filters=%7B%22text%22%3A%22true%22%2C%22type%22%3A%22auto%22%2C%22highlights%22%3A%22true%22%7D)
</Info>
