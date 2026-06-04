> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Contents

> Get the full page contents, summaries, and metadata for a list of URLs.

Returns instant results from our cache, with automatic live crawling as fallback for uncached pages.

***

<Card title="Get your Exa API key" icon="key" horizontal href="https://dashboard.exa.ai/api-keys" />


## OpenAPI

````yaml post /contents
openapi: 3.1.0
info:
  title: Exa Public API
  version: 2.0.0
servers:
  - url: https://api.exa.ai
security:
  - apiKey: []
  - bearer: []
tags: []
paths:
  /contents:
    post:
      summary: Contents
      operationId: getContents
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ContentsRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              example:
                requestId: e492118ccdedcba5088bfc4357a8a125
                results:
                  - title: A Comprehensive Overview of Large Language Models
                    url: https://arxiv.org/pdf/2307.06435.pdf
                    publishedDate: '2023-11-16T01:36:32.547Z'
                    author: >-
                      Humza  Naveed, University of Engineering and Technology
                      (UET), Lahore, Pakistan
                    id: https://arxiv.org/abs/2307.06435
                    image: https://arxiv.org/pdf/2307.06435.pdf/page_1.png
                    favicon: https://arxiv.org/favicon.ico
                    text: >-
                      Abstract Large Language Models (LLMs) have recently
                      demonstrated remarkable capabilities...
                    highlights:
                      - Such requirements have limited their adoption...
                    highlightScores:
                      - 0.4600165784358978
                    summary: >-
                      This overview paper on Large Language Models (LLMs)
                      highlights key developments...
                    subpages:
                      - id: https://arxiv.org/abs/2303.17580
                        url: https://arxiv.org/pdf/2303.17580.pdf
                        title: >-
                          HuggingGPT: Solving AI Tasks with ChatGPT and its
                          Friends in Hugging Face
                        author: >-
                          Yongliang  Shen, Microsoft Research Asia, Kaitao 
                          Song, Microsoft Research Asia, Xu  Tan, Microsoft
                          Research Asia, Dongsheng  Li, Microsoft Research Asia,
                          Weiming  Lu, Microsoft Research Asia, Yueting  Zhuang,
                          Microsoft Research Asia, yzhuang@zju.edu.cn, Zhejiang 
                          University, Microsoft Research Asia, Microsoft 
                          Research, Microsoft Research Asia
                        publishedDate: '2023-11-16T01:36:20.486Z'
                        text: >-
                          HuggingGPT: Solving AI Tasks with ChatGPT and its
                          Friends in Hugging Face Date Published: 2023-05-25
                          Authors: Yongliang Shen, Microsoft Research Asia
                          Kaitao Song, Microsoft Research Asia Xu Tan, Microsoft
                          Research Asia Dongsheng Li, Microsoft Research Asia
                          Weiming Lu, Microsoft Research Asia Yueting Zhuang,
                          Microsoft Research Asia, yzhuang@zju.edu.cn Zhejiang
                          University, Microsoft Research Asia Microsoft
                          Research, Microsoft Research Asia Abstract Solving
                          complicated AI tasks with different domains and
                          modalities is a key step toward artificial general
                          intelligence. While there are abundant AI models
                          available for different domains and modalities, they
                          cannot handle complicated AI tasks. Considering large
                          language models (LLMs) have exhibited exceptional
                          ability in language understanding, generation,
                          interaction, and reasoning, we advocate that LLMs
                          could act as a controller to manage existing AI models
                          to solve complicated AI tasks and language could be a
                          generic interface to empower t
                        summary: >-
                          HuggingGPT is a framework using ChatGPT as a central
                          controller to orchestrate various AI models from
                          Hugging Face to solve complex tasks. ChatGPT plans the
                          task, selects appropriate models based on their
                          descriptions, executes subtasks, and summarizes the
                          results. This approach addresses limitations of LLMs
                          by allowing them to handle multimodal data (vision,
                          speech) and coordinate multiple models for complex
                          tasks, paving the way for more advanced AI systems.
                        highlights:
                          - >-
                            2) Recently, some researchers started to investigate
                            the integration of using tools or models in LLMs  .
                        highlightScores:
                          - 0.32679107785224915
                    extras:
                      links: []
                context: <string>
                statuses:
                  - id: https://example.com
                    status: success
                    source: cached
                costDollars:
                  total: 0.007
                  search:
                    neural: 0.007
              schema:
                $ref: '#/components/schemas/ContentsResponse'
      x-codeSamples:
        - lang: bash
          label: Simple contents retrieval
          source: |-
            curl -X POST 'https://api.exa.ai/contents' \
              -H 'x-api-key: YOUR-EXA-API-KEY' \
              -H 'Content-Type: application/json' \
              -d '{
                "urls": ["https://arxiv.org/abs/2307.06435"],
                "text": true
              }'
        - lang: python
          label: Simple contents retrieval
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.get_contents(
                urls=["https://arxiv.org/abs/2307.06435"],
                text=True
            )

            print(results)
        - lang: javascript
          label: Simple contents retrieval
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.getContents(
                ["https://arxiv.org/abs/2307.06435"],
                { text: true }
            );

            console.log(results);
        - lang: bash
          label: Advanced contents retrieval
          source: |-
            curl --request POST \
              --url https://api.exa.ai/contents \
              --header 'x-api-key: YOUR-EXA-API-KEY' \
              --header 'Content-Type: application/json' \
              --data '{
                "urls": ["https://arxiv.org/abs/2307.06435"],
                "text": {
                  "maxCharacters": 1000,
                  "includeHtmlTags": false
                },
                "highlights": {
                  "query": "Key findings"
                },
                "summary": {
                  "query": "Main research contributions"
                },
                "subpages": 1,
                "subpageTarget": "references",
                "extras": {
                  "links": 2,
                  "imageLinks": 1
                }
              }'
        - lang: python
          label: Advanced contents retrieval
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.get_contents(
                urls=["https://arxiv.org/abs/2307.06435"],
                text={
                    "maxCharacters": 1000,
                    "includeHtmlTags": False
                },
                highlights={
                    "query": "Key findings"
                },
                summary={
                    "query": "Main research contributions"
                },
                subpages=1,
                subpage_target="references",
                extras={
                    "links": 2,
                    "image_links": 1
                }
            )

            print(results)
        - lang: javascript
          label: Advanced contents retrieval
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.getContents(
                ["https://arxiv.org/abs/2307.06435"],
                {
                    text: {
                        maxCharacters: 1000,
                        includeHtmlTags: false
                    },
                    highlights: {
                        query: "Key findings"
                    },
                    summary: {
                        query: "Main research contributions"
                    },
                    subpages: 1,
                    subpageTarget: "references",
                    extras: {
                        links: 2,
                        imageLinks: 1
                    }
                }
            );

            console.log(results);
components:
  schemas:
    ContentsRequest:
      allOf:
        - type: object
          properties:
            ids:
              minItems: 1
              maxItems: 100
              type: array
              items:
                type: string
                minLength: 1
                maxLength: 2048
              description: Array of document IDs obtained from searches.
              example:
                - https://arxiv.org/pdf/2307.06435
            urls:
              minItems: 1
              maxItems: 100
              type: array
              items:
                type: string
                minLength: 1
                maxLength: 2048
              description: >-
                Array of URLs to crawl (backwards compatible with 'ids'
                parameter).
              example:
                - https://arxiv.org/pdf/2307.06435
            compliance:
              anyOf:
                - type: string
                  enum:
                    - hipaa
                  description: >-
                    Enterprise-only compliance mode. Set to `hipaa` to require
                    HIPAA-safe processing. Requests fail closed or restrict
                    features when the requested behavior requires non-HIPAA-safe
                    processors.
                  example: hipaa
                - type: 'null'
          required:
            - urls
        - $ref: '#/components/schemas/ContentsOptions'
    ContentsResponse:
      type: object
      properties:
        requestId:
          type: string
          description: Unique identifier for the request.
          example: b5947044c4b78efa9552a7c89b306d95
        results:
          type: array
          items:
            $ref: '#/components/schemas/SearchResultOutput'
        context:
          type: string
          description: >-
            Deprecated. Combined context string from search results. Use
            highlights or text instead.
          deprecated: true
        statuses:
          description: Status information for each requested URL or document ID.
          type: array
          items:
            type: object
            properties:
              id:
                type: string
                description: The URL or document ID that was requested.
                example: https://example.com
              status:
                type: string
                enum:
                  - success
                  - error
                description: Status of the content fetch operation.
                example: success
              source:
                description: Where the returned content was sourced from.
                type: string
                enum:
                  - cached
                  - crawled
              error:
                anyOf:
                  - type: object
                    properties:
                      tag:
                        description: Specific error type.
                        example: CRAWL_NOT_FOUND
                        type: string
                      httpStatusCode:
                        anyOf:
                          - type: integer
                            minimum: 100
                            maximum: 599
                          - type: 'null'
                        description: The corresponding HTTP status code.
                        example: 404
                    additionalProperties: false
                  - type: 'null'
                description: Error details, only present when status is "error".
            required:
              - id
              - status
            additionalProperties: false
        costDollars:
          $ref: '#/components/schemas/CostDollarsOutput'
      additionalProperties: false
    ContentsOptions:
      type: object
      properties:
        text:
          anyOf:
            - description: Text extraction options for each result.
              oneOf:
                - type: boolean
                  title: Simple text retrieval
                  description: >-
                    If true, returns full page text with default settings. If
                    false, disables text return.
                  default: false
                - type: object
                  properties:
                    maxCharacters:
                      anyOf:
                        - type: integer
                          minimum: 1
                          maximum: 10000
                          description: >-
                            Maximum character limit for the full page text.
                            Useful for controlling response size and API costs.
                            Maximum supported value is 10000.
                          example: 1000
                        - type: 'null'
                    includeHtmlTags:
                      anyOf:
                        - type: boolean
                          description: >-
                            If true, include lightweight HTML tags in returned
                            text instead of plain markdown-style text. Use
                            maxAgeHours: 0 when you need this applied to freshly
                            fetched content.
                          example: false
                          default: false
                        - type: 'null'
                    verbosity:
                      anyOf:
                        - type: string
                          enum:
                            - compact
                            - standard
                            - full
                          description: >-
                            Controls text rendering verbosity. compact focuses
                            on main content, standard includes more surrounding
                            page context, and full requests the most complete
                            rendered text. Some pages may produce identical
                            standard and full output. Use maxAgeHours: 0 when
                            you need this applied to freshly fetched content.
                          example: standard
                          default: compact
                        - type: 'null'
                    includeSections:
                      anyOf:
                        - type: array
                          items:
                            type: string
                            enum:
                              - header
                              - navigation
                              - banner
                              - body
                              - sidebar
                              - footer
                              - metadata
                          description: >-
                            Best-effort. Only include content classified into
                            these semantic page sections. Section classification
                            may be unavailable or incomplete for some pages;
                            validate output if strict filtering is required. Use
                            maxAgeHours: 0 when you need this applied to freshly
                            fetched content.
                          example:
                            - body
                            - header
                        - type: 'null'
                    excludeSections:
                      anyOf:
                        - type: array
                          items:
                            type: string
                            enum:
                              - header
                              - navigation
                              - banner
                              - body
                              - sidebar
                              - footer
                              - metadata
                          description: >-
                            Exclude content classified into these semantic page
                            sections. Section classification is best-effort. Use
                            maxAgeHours: 0 when you need this applied to freshly
                            fetched content.
                          example:
                            - navigation
                            - footer
                            - sidebar
                        - type: 'null'
                  title: Advanced text options
                  description: >-
                    Advanced options for controlling text extraction. Use this
                    when you need to limit text length or include HTML
                    structure.
            - type: 'null'
        highlights:
          anyOf:
            - description: >-
                Text snippets the LLM identifies as most relevant from each
                page.
              oneOf:
                - type: boolean
                  title: Simple highlights retrieval
                  description: >-
                    If true, returns highlights with default settings. If false,
                    disables highlights.
                  default: false
                - type: object
                  properties:
                    query:
                      anyOf:
                        - type: string
                          description: >-
                            Custom query that guides which highlights the LLM
                            picks.
                          example: Key advancements
                        - type: 'null'
                    maxCharacters:
                      anyOf:
                        - type: integer
                          minimum: 1
                          maximum: 10000
                          description: >-
                            Maximum number of characters to return for
                            highlights. Controls the total length of highlight
                            text returned per URL. Maximum supported value is
                            10000.
                          example: 2000
                        - type: 'null'
                    numSentences:
                      anyOf:
                        - type: integer
                          minimum: 1
                          description: >-
                            Deprecated and will be removed in a future release.
                            Currently mapped to a character budget of about 1333
                            characters per sentence. Pass highlights: true for
                            default highlights, or { query } to guide selection
                            with your own query.
                          example: 1
                          deprecated: true
                        - type: 'null'
                    highlightsPerUrl:
                      anyOf:
                        - type: integer
                          minimum: 1
                          description: >-
                            Deprecated and will be removed in a future release.
                            Currently ignored. Pass highlights: true for default
                            highlights, or { query } to guide selection with
                            your own query.
                          example: 1
                          deprecated: true
                        - type: 'null'
                  title: Advanced highlights options
                  description: >-
                    Advanced options for steering highlight extraction. Pass
                    highlights: true for the highest-quality default; supply
                    this object only when you need to guide selection with your
                    own query.
            - type: 'null'
        summary:
          anyOf:
            - type: object
              properties:
                query:
                  anyOf:
                    - type: string
                      description: Custom query for the LLM-generated summary.
                      example: Main developments
                    - type: 'null'
                schema:
                  anyOf:
                    - type: object
                      propertyNames:
                        type: string
                      additionalProperties:
                        $ref: '#/components/schemas/JsonValue'
                      description: >-
                        JSON schema for structured output from summary. See
                        https://json-schema.org/overview/what-is-jsonschema for
                        JSON Schema documentation.
                      example:
                        $schema: http://json-schema.org/draft-07/schema#
                        title: Title
                        type: object
                        properties:
                          Property 1:
                            type: string
                            description: Description
                          Property 2:
                            type: string
                            enum:
                              - option 1
                              - option 2
                              - option 3
                            description: Description
                        required:
                          - Property 1
                    - type: 'null'
              description: Summary of the webpage.
            - type: 'null'
        extras:
          anyOf:
            - type: object
              properties:
                links:
                  anyOf:
                    - type: integer
                      minimum: 0
                      maximum: 1000
                      description: Number of URLs to return from each webpage.
                      example: 1
                      default: 0
                    - type: 'null'
                imageLinks:
                  anyOf:
                    - type: integer
                      minimum: 0
                      maximum: 1000
                      description: Number of images to return for each result.
                      example: 1
                      default: 0
                    - type: 'null'
                richImageLinks:
                  anyOf:
                    - type: integer
                      minimum: 0
                      maximum: 1000
                      description: Number of rich image links to return for each result.
                      default: 0
                    - type: 'null'
                richLinks:
                  anyOf:
                    - type: integer
                      minimum: 0
                      maximum: 1000
                      description: Number of rich links to return for each result.
                      default: 0
                    - type: 'null'
                codeBlocks:
                  anyOf:
                    - type: integer
                      minimum: 0
                      maximum: 1000
                      description: Number of code blocks to return for each result.
                      default: 0
                    - type: 'null'
              description: Extra parameters to pass.
            - type: 'null'
        context:
          anyOf:
            - description: >-
                Deprecated: Use highlights or text instead. Returns page
                contents as a combined context string.
              deprecated: true
              oneOf:
                - type: boolean
                  description: >-
                    Deprecated: Use highlights or text instead. Returns page
                    contents as a combined context string.
                  example: true
                  deprecated: true
                - type: object
                  properties:
                    maxCharacters:
                      type: integer
                      minimum: 1
                      maximum: 10000
                      description: >-
                        Deprecated. Maximum character limit for the context
                        string. Maximum supported value is 10000.
                      example: 10000
                  description: >-
                    Deprecated: Use highlights or text instead. Returns page
                    contents as a combined context string.
                  deprecated: true
            - type: 'null'
        livecrawl:
          anyOf:
            - type: string
              enum:
                - never
                - always
                - fallback
                - preferred
              description: >-
                Deprecated: Use maxAgeHours instead for content freshness
                control. livecrawl does not guarantee freshly fetched parser
                output and may be served according to server freshness policy.
                Do not send livecrawl and maxAgeHours together.
              example: preferred
              deprecated: true
            - type: 'null'
        livecrawlTimeout:
          anyOf:
            - type: integer
              exclusiveMinimum: 0
              maximum: 90000
              description: The timeout for livecrawling in milliseconds.
              example: 1000
              default: 10000
            - type: 'null'
        maxAgeHours:
          anyOf:
            - type: integer
              minimum: -1
              maximum: 720
              description: >-
                Maximum age of cached content in hours. Positive values use
                cached content if it is less than this many hours old; 0 fetches
                fresh content and is the supported way to apply text rendering
                options to newly fetched pages; -1 always uses cache; omitted
                uses fallback fetching when cached content is unavailable.
                Maximum supported value is 720 hours.
              example: 24
            - type: 'null'
        subpages:
          anyOf:
            - type: integer
              minimum: 0
              maximum: 100
              description: >-
                The number of subpages to crawl. The actual number crawled may
                be limited by system constraints.
              example: 1
              default: 0
            - type: 'null'
        subpageTarget:
          anyOf:
            - description: >-
                Term to find specific subpages of search results. Can be a
                single string or an array of strings.
              example: sources
              oneOf:
                - type: string
                  minLength: 1
                  maxLength: 100
                - minItems: 0
                  maxItems: 100
                  type: array
                  items:
                    type: string
                    minLength: 1
                    maxLength: 100
            - type: 'null'
    SearchResultOutput:
      type: object
      properties:
        title:
          type: string
          description: The title of the search result.
          example: A Comprehensive Overview of Large Language Models
        url:
          type: string
          description: The URL of the search result.
          example: https://arxiv.org/pdf/2307.06435.pdf
          format: uri
        publishedDate:
          description: >-
            An estimate of the creation date, from parsing HTML content. Format
            is YYYY-MM-DD.
          example: '2023-11-16T01:36:32.547Z'
          format: date-time
          type: string
        author:
          description: If available, the author of the content.
          example: Humza Naveed
          anyOf:
            - type: string
            - type: 'null'
        id:
          description: >-
            The temporary ID for the document. Useful for the /contents
            endpoint.
          example: https://arxiv.org/abs/2307.06435
          type: string
        image:
          description: The URL of an image associated with the search result, if available.
          example: https://arxiv.org/pdf/2307.06435.pdf/page_1.png
          format: uri
          type: string
        favicon:
          description: The URL of the favicon for the search result's domain.
          example: https://arxiv.org/favicon.ico
          format: uri
          type: string
        text:
          description: The full content text of the search result.
          example: >-
            Abstract Large Language Models (LLMs) have recently demonstrated
            remarkable capabilities...
          type: string
        highlights:
          description: Array of highlights extracted from the search result content.
          example:
            - Such requirements have limited their adoption...
          type: array
          items:
            type: string
        highlightScores:
          description: Array of cosine similarity scores for each highlighted snippet.
          example:
            - 0.4600165784358978
          type: array
          items:
            type: number
            format: float
        summary:
          description: Summary of the webpage.
          example: >-
            This overview paper on Large Language Models (LLMs) highlights key
            developments...
          type: string
        subpages:
          description: Array of subpages for the search result.
          type: array
          items:
            type: object
            properties:
              title:
                type: string
                description: The title of the search result.
                example: A Comprehensive Overview of Large Language Models
              url:
                type: string
                description: The URL of the search result.
                example: https://arxiv.org/pdf/2307.06435.pdf
                format: uri
              publishedDate:
                description: >-
                  An estimate of the creation date, from parsing HTML content.
                  Format is YYYY-MM-DD.
                example: '2023-11-16T01:36:32.547Z'
                format: date-time
                type: string
              author:
                description: If available, the author of the content.
                example: Humza Naveed
                anyOf:
                  - type: string
                  - type: 'null'
              id:
                description: >-
                  The temporary ID for the document. Useful for the /contents
                  endpoint.
                example: https://arxiv.org/abs/2307.06435
                type: string
              image:
                description: >-
                  The URL of an image associated with the search result, if
                  available.
                example: https://arxiv.org/pdf/2307.06435.pdf/page_1.png
                format: uri
                type: string
              favicon:
                description: The URL of the favicon for the search result's domain.
                example: https://arxiv.org/favicon.ico
                format: uri
                type: string
            required:
              - title
              - url
            additionalProperties: false
        entities:
          description: >-
            Structured entity data for company or person search results. Only
            returned for category=company or category=people searches.
          type: array
          items:
            oneOf:
              - type: object
                properties:
                  id:
                    type: string
                    description: Stable company entity identifier.
                  type:
                    type: string
                    const: company
                    description: Entity discriminator.
                  version:
                    type: integer
                    minimum: 1
                    description: Entity schema version.
                  properties:
                    type: object
                    properties:
                      name:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Company name.
                      foundedYear:
                        anyOf:
                          - type: integer
                          - type: 'null'
                        description: Year the company was founded.
                      description:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Short company description.
                      workforce:
                        anyOf:
                          - type: object
                            properties:
                              total:
                                anyOf:
                                  - type: number
                                  - type: 'null'
                                description: Total estimated employee count.
                            required:
                              - total
                            additionalProperties: false
                          - type: 'null'
                        description: Company workforce information.
                      headquarters:
                        anyOf:
                          - type: object
                            properties:
                              address:
                                anyOf:
                                  - type: string
                                  - type: 'null'
                                description: Company headquarters street address.
                              city:
                                anyOf:
                                  - type: string
                                  - type: 'null'
                                description: Company headquarters city.
                              postalCode:
                                anyOf:
                                  - type: string
                                  - type: 'null'
                                description: Company headquarters postal code.
                              country:
                                anyOf:
                                  - type: string
                                  - type: 'null'
                                description: Company headquarters country.
                            required:
                              - address
                              - city
                              - postalCode
                              - country
                            additionalProperties: false
                          - type: 'null'
                        description: Company headquarters information.
                      financials:
                        anyOf:
                          - type: object
                            properties:
                              revenueAnnual:
                                anyOf:
                                  - type: number
                                  - type: 'null'
                                description: Estimated annual revenue in USD.
                              fundingTotal:
                                anyOf:
                                  - type: number
                                  - type: 'null'
                                description: Total funding raised in USD.
                              fundingLatestRound:
                                anyOf:
                                  - type: object
                                    properties:
                                      name:
                                        anyOf:
                                          - type: string
                                          - type: 'null'
                                        description: Funding round name.
                                      date:
                                        anyOf:
                                          - type: string
                                          - type: 'null'
                                        description: Funding round date.
                                      amount:
                                        anyOf:
                                          - type: number
                                          - type: 'null'
                                        description: Funding round amount in USD.
                                    required:
                                      - name
                                      - date
                                      - amount
                                    additionalProperties: false
                                  - type: 'null'
                                description: Most recent funding round, when available.
                            required:
                              - revenueAnnual
                              - fundingTotal
                              - fundingLatestRound
                            additionalProperties: false
                          - type: 'null'
                        description: Company financial information.
                      webTraffic:
                        anyOf:
                          - type: object
                            properties:
                              visitsMonthly:
                                anyOf:
                                  - type: number
                                  - type: 'null'
                                description: Estimated monthly website visits.
                              countryRank:
                                anyOf:
                                  - type: integer
                                  - type: 'null'
                                description: >-
                                  Estimated website traffic rank within the
                                  company's primary country.
                              avgDurationSeconds:
                                anyOf:
                                  - type: number
                                  - type: 'null'
                                description: Estimated average visit duration, in seconds.
                              history:
                                type: array
                                items:
                                  type: object
                                  properties:
                                    value:
                                      type: number
                                      description: >-
                                        Estimated monthly visits for this
                                        period.
                                    dateFrom:
                                      type: string
                                      description: >-
                                        Start month for this value, formatted as
                                        YYYY-MM.
                                    dateTo:
                                      type: string
                                      description: >-
                                        End month for this value, formatted as
                                        YYYY-MM.
                                  required:
                                    - value
                                    - dateFrom
                                    - dateTo
                                  additionalProperties: false
                                description: Historical monthly website visits.
                            required:
                              - visitsMonthly
                              - countryRank
                              - avgDurationSeconds
                              - history
                            additionalProperties: false
                          - type: 'null'
                        description: Company web traffic information.
                    required:
                      - name
                      - foundedYear
                      - description
                      - workforce
                      - headquarters
                      - financials
                      - webTraffic
                    additionalProperties: false
                    description: Company-specific entity fields.
                required:
                  - id
                  - type
                  - version
                  - properties
                additionalProperties: false
              - type: object
                properties:
                  id:
                    type: string
                    description: Stable person entity identifier.
                  type:
                    type: string
                    const: person
                    description: Entity discriminator.
                  version:
                    type: integer
                    minimum: 1
                    description: Entity schema version.
                  properties:
                    type: object
                    properties:
                      name:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Person name.
                      firstName:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Person first name.
                      lastName:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Person last name.
                      location:
                        anyOf:
                          - type: string
                          - type: 'null'
                        description: Person location.
                      workHistory:
                        type: array
                        items:
                          type: object
                          properties:
                            title:
                              anyOf:
                                - type: string
                                - type: 'null'
                              description: Role title.
                            location:
                              anyOf:
                                - type: string
                                - type: 'null'
                              description: Role location.
                            dates:
                              anyOf:
                                - type: object
                                  properties:
                                    from:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Start date for the date range.
                                    to:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: End date for the date range.
                                  required:
                                    - from
                                    - to
                                  additionalProperties: false
                                - type: 'null'
                              description: Role date range.
                            company:
                              anyOf:
                                - type: object
                                  properties:
                                    id:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Referenced company identifier.
                                    name:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Referenced company name.
                                  required:
                                    - id
                                    - name
                                  additionalProperties: false
                                - type: 'null'
                              description: Company for this role.
                          required:
                            - title
                            - location
                            - dates
                            - company
                          additionalProperties: false
                        description: Known professional roles for this person.
                      educationHistory:
                        type: array
                        items:
                          type: object
                          properties:
                            degree:
                              anyOf:
                                - type: string
                                - type: 'null'
                              description: Degree or credential.
                            dates:
                              anyOf:
                                - type: object
                                  properties:
                                    from:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Start date for the date range.
                                    to:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: End date for the date range.
                                  required:
                                    - from
                                    - to
                                  additionalProperties: false
                                - type: 'null'
                              description: Education date range.
                            institution:
                              anyOf:
                                - type: object
                                  properties:
                                    id:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Referenced institution identifier.
                                    name:
                                      anyOf:
                                        - type: string
                                        - type: 'null'
                                      description: Referenced institution name.
                                  required:
                                    - id
                                    - name
                                  additionalProperties: false
                                - type: 'null'
                              description: Education institution.
                          required:
                            - degree
                            - dates
                            - institution
                          additionalProperties: false
                        description: Known education history for this person.
                    required:
                      - name
                      - firstName
                      - lastName
                      - location
                      - workHistory
                      - educationHistory
                    additionalProperties: false
                    description: Person-specific entity fields.
                required:
                  - id
                  - type
                  - version
                  - properties
                additionalProperties: false
            type: object
        extras:
          description: Results from extras.
          example:
            links: []
          type: object
          properties:
            links:
              description: Array of links from the search result.
              example: []
              type: array
              items:
                type: string
          additionalProperties: false
      required:
        - title
        - url
      additionalProperties: false
    CostDollarsOutput:
      type: object
      properties:
        total:
          description: >-
            Estimated total dollar cost for the completed request. This response
            value is not an invoice record.
          example: 0.007
          format: float
          type: number
        search:
          description: >-
            Endpoint-dependent estimated search cost breakdown by retrieval
            mode. Instant, fast, and auto search responses may include neural
            search cost. Deep search modes may be reflected only in total.
          type: object
          properties:
            neural:
              description: Cost of neural search operations.
              example: 0.007
              format: float
              type: number
          additionalProperties: false
      additionalProperties: false
      description: >-
        Endpoint-dependent estimated dollar cost breakdown for the completed
        request. Billing is computed from usage counters rather than this
        response object.
    JsonValue:
      description: Any JSON value.
      oneOf:
        - type: 'null'
        - type: boolean
        - type: number
        - type: string
        - type: array
          items:
            $ref: '#/components/schemas/JsonValue'
        - type: object
          propertyNames:
            type: string
          additionalProperties:
            $ref: '#/components/schemas/JsonValue'
  securitySchemes:
    apiKey:
      type: apiKey
      name: x-api-key
      in: header
      description: >-
        Pass your Exa API key in the x-api-key header. You can also authenticate
        with Authorization: Bearer <key>.
    bearer:
      type: http
      scheme: bearer
      description: >-
        Pass your Exa API key in the x-api-key header. You can also authenticate
        with Authorization: Bearer <key>.

````