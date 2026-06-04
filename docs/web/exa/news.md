> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# News Search

Exa's news search provides real-time access to a continuously updated index of news sources including major publications, trade press, and niche outlets. It supports semantic search with native date filtering to surface the freshest, most relevant results.

## When to Use

If you're building:

* A **finance or investment research platform**
  * Monitor market-moving news, earnings announcements, and sector developments in real time
  * "Tesla quarterly earnings results"
  * "semiconductor supply chain disruptions"
* A **cybersecurity or threat intelligence tool**
  * Track vulnerability disclosures, breach reports, and emerging threat actor activity
  * "zero-day vulnerabilities disclosed this week"
* A **GTM intelligence or competitive monitoring workflow**
  * Surface press coverage, product launches, and funding announcements for target accounts
  * "OpenAI product launches"
  * "AI startup funding announcements"
* A **consulting or enterprise research application**
  * Gather and summarize current news sources for client briefings and industry reports
  * "trade policy changes US China"

## Basic Usage

<CodeGroup>
  ```bash curl theme={null}
  curl -X POST https://api.exa.ai/search \
    -H "x-api-key: YOUR_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
      "query": "AI regulation updates in the European Union",
      "type": "auto",
      "numResults": 10
    }'
  ```

  ```python python theme={null}
  from exa_py import Exa

  exa = Exa(api_key="YOUR_API_KEY")

  results = exa.search(
      "AI regulation updates in the European Union",
      type="auto",
      num_results=10
  )
  ```

  ```javascript javascript theme={null}
  import Exa from "exa-js";

  const exa = new Exa("YOUR_API_KEY");

  const results = await exa.search(
    "AI regulation updates in the European Union",
    {
      type: "auto",
      numResults: 10,
    }
  );
  ```
</CodeGroup>

<Info>
  Try it now: [News Search in the API Playground →](https://dashboard.exa.ai/playground/search?q=AI+regulation+updates\&c=news\&filters=%7B%22highlights%22%3A%22true%22%7D)
</Info>
