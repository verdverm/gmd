> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Code Search

Exa's code search indexes billions of GitHub repositories, documentation pages, and Stack Overflow posts, using semantic search to match natural language queries to real, working code examples.

Read the blog post: [WebCode: Search Evals for Coding Agents](https://exa.ai/blog/webcode)

## When to Use

If you're building:

* An **AI coding agent or code-generation tool**
  * Ground model outputs with real, up-to-date code examples and API references
  * "how to use the Vercel AI SDK streaming API"
  * "correct syntax for Next.js 14 app router with TypeScript"
* A **developer documentation or search platform**
  * Surface working code snippets from across GitHub, Stack Overflow, and docs sites
  * "pandas dataframe filtering and groupby operations"
* An **AI infrastructure or agent framework**
  * Give agents reliable web context for code tasks, reducing hallucinated imports and outdated syntax
  * "how to set up a reproducible Nix Rust development environment"
* A **developer productivity tool**
  * Help engineers find configuration patterns, migration guides, and setup recipes
  * "Docker Compose for PostgreSQL and Redis"

## Basic Usage

<CodeGroup>
  ```bash curl theme={null}
  curl -X POST https://api.exa.ai/search \
    -H "x-api-key: YOUR_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
      "query": "how to use Exa search in python",
      "type": "fast",
      "numResults": 10,
      "contents": {
        "highlights": true
      }
    }'
  ```

  ```python python theme={null}
  from exa_py import Exa

  exa = Exa(api_key="YOUR_API_KEY")

  results = exa.search(
      "how to use Exa search in python",
      type="fast",
      num_results=10,
      contents={"highlights": True},
  )
  ```

  ```javascript javascript theme={null}
  import Exa from "exa-js";

  const exa = new Exa("YOUR_API_KEY");

  const results = await exa.search(
    "how to use Exa search in python",
    {
      type: "fast",
      numResults: 10,
      contents: {
        highlights: true,
      },
    }
  );
  ```
</CodeGroup>

<Info>
  Try it now: [Code Search in the API Playground →](https://dashboard.exa.ai/playground/search?q=how+to+use+Exa+search+in+python\&filters=%7B%22type%22%3A%22fast%22%2C%22highlights%22%3A%22true%22%7D)
</Info>
