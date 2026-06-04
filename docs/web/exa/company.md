> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Company Search

Exa's company search indexes 50M+ companies, updated weekly, and supports natural language queries across industry, funding stage, headcount, geography, and technology.

Read the blog post: [Introducing Exa's Company Search Benchmark](https://exa.ai/blog/company-search-benchmark)

<Tip>
  For agentic list-building and enrichment out of the box, use [Websets →](/websets/api-guide).
</Tip>

## When to Use

If you're building:

* A **GTM intelligence or lead-generation platform**
  * Build targeted company lists by industry, headcount, geography, and funding stage
  * "Series A fintech companies in Switzerland with 50–200 employees"
  * "Japanese AI companies founded in 2023"
* A **finance or investment research tool**
  * Source deals, map competitive landscapes, and track funding rounds
  * "agtech companies that raised Series A in the US"
  * "startups that raised 30M to 80M"
* A **competitive intelligence or market-mapping workflow**
  * Discover emerging players and alternatives in a space
  * "companies like Stripe"
  * "competitors of Notion"
* A **consulting or professional services engagement**
  * Research industries, identify vendors, and build market scans for client deliverables
  * "German enterprise SaaS companies with more than 500 employees"

## Basic Usage

<CodeGroup>
  ```bash curl theme={null}
  curl -X POST https://api.exa.ai/search \
    -H "x-api-key: YOUR_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
      "query": "Agtech companies optimizing pesticide placement with computer vision",
      "category": "company",
      "type": "auto",
      "numResults": 10
    }'
  ```

  ```python python theme={null}
  from exa_py import Exa

  exa = Exa(api_key="YOUR_API_KEY")

  results = exa.search(
      "Agtech companies optimizing pesticide placement with computer vision",
      category="company",
      type="auto",
      num_results=10
  )
  ```

  ```javascript javascript theme={null}
  import Exa from "exa-js";

  const exa = new Exa("YOUR_API_KEY");

  const results = await exa.search(
    "Agtech companies optimizing pesticide placement with computer vision",
    {
      category: "company",
      type: "auto",
      numResults: 10,
    }
  );
  ```
</CodeGroup>

## Structured Entity Metadata

Company Search returns structured company metadata in `entities` for result rows that resolve to a company. Each company entity has `type: "company"`, a stable `id`, a schema `version`, and a `properties` object with company profile fields.

```json theme={null}
{
  "results": [
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
  ]
}
```

<Info>
  Try it now: [Company Search in the API Playground →](https://dashboard.exa.ai/playground/search?q=fintech%20companies%20in%20Switzerland\&c=company\&filters=%7B%22text%22%3A%22true%22%2C%22type%22%3A%22auto%22%2C%22highlights%22%3A%22true%22%7D)
</Info>
