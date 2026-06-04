> ## Documentation Index
> Fetch the complete documentation index at: https://exa.ai/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Search

> The search endpoint lets you search the web and extract contents from the results.

<Card title="Get your Exa API key" icon="key" horizontal href="https://dashboard.exa.ai/api-keys" />


## OpenAPI

````yaml post /search
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
  /search:
    post:
      summary: Search
      description: >-
        Perform a search with an Exa prompt-engineered query and retrieve a list
        of relevant results. Optionally get contents.
      operationId: search
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SearchRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              example:
                requestId: b5947044c4b78efa9552a7c89b306d95
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
                resolvedSearchType: ''
                context: <string>
                output:
                  content: <string>
                  grounding:
                    - field: <string>
                      citations:
                        - url: <string>
                          title: <string>
                      confidence: high
                costDollars:
                  total: 0.007
                  search:
                    neural: 0.007
              schema:
                $ref: '#/components/schemas/SearchResponse'
            text/event-stream:
              schema:
                $ref: '#/components/schemas/SearchStreamChunk'
      x-codeSamples:
        - lang: bash
          label: Simple search with contents
          source: |-
            curl -X POST 'https://api.exa.ai/search' \
              -H 'x-api-key: YOUR-EXA-API-KEY' \
              -H 'Content-Type: application/json' \
              -d '{
                "query": "Latest research in LLMs",
                "contents": {
                  "highlights": true
                }
              }'
        - lang: python
          label: Simple search with contents
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.search(
                "Latest research in LLMs",
                contents={"highlights": True}
            )

            print(results)
        - lang: javascript
          label: Simple search with contents
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.search(
                'Latest research in LLMs',
                { contents: { highlights: true } }
            );

            console.log(results);
        - lang: bash
          label: Advanced search with filters
          source: |-
            curl --request POST \
              --url https://api.exa.ai/search \
              --header 'x-api-key: <token>' \
              --header 'Content-Type: application/json' \
              --data '{
                "query": "Latest research in LLMs",
                "type": "auto",
                "category": "research paper",
                "numResults": 10,
                "moderation": true,
                "contents": {
                  "text": true,
                  "summary": {
                    "query": "Main developments"
                  },
                  "subpages": 1,
                  "subpageTarget": "sources",
                  "extras": {
                    "links": 1,
                    "imageLinks": 1
                  }
                }
              }'
        - lang: bash
          label: Deep search with query variations
          source: |-
            curl --request POST \
              --url https://api.exa.ai/search \
              --header 'x-api-key: <token>' \
              --header 'Content-Type: application/json' \
              --data '{
                "query": "Who is the CEO of OpenAI?",
                "additionalQueries": [
                  "OpenAI CEO current",
                  "OpenAI leadership official source"
                ],
                "type": "deep",
                "systemPrompt": "Prefer official sources and avoid duplicate results",
                "outputSchema": {
                  "type": "object",
                  "properties": {
                    "leader": { "type": "string" },
                    "title": { "type": "string" },
                    "sourceCount": { "type": "number" }
                  },
                  "required": ["leader", "title"]
                },
                "contents": {
                  "text": true
                }
              }'
        - lang: python
          label: Advanced search with filters
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.search(
                "Latest research in LLMs",
                type="auto",
                category="research paper",
                num_results=10,
                moderation=True,
                contents={
                    "text": True,
                    "summary": {
                        "query": "Main developments"
                    },
                    "subpages": 1,
                    "subpage_target": "sources",
                    "extras": {
                        "links": 1,
                        "image_links": 1
                    }
                },
            )

            print(results)
        - lang: javascript
          label: Advanced search with filters
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.search('Latest research in LLMs', {
                type: 'auto',
                category: 'research paper',
                numResults: 10,
                moderation: true,
                contents: {
                    text: true,
                    summary: {
                        query: 'Main developments'
                    },
                    subpages: 1,
                    subpageTarget: 'sources',
                    extras: {
                        links: 1,
                        imageLinks: 1
                    }
                }
            });

            console.log(results);
        - lang: python
          label: Deep search with query variations
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.search(
                "Who is the CEO of OpenAI?",
                additional_queries=[
                    "OpenAI CEO current",
                    "OpenAI leadership official source"
                ],
                type="deep",
                system_prompt="Prefer official sources and avoid duplicate results",
                output_schema={
                    "type": "object",
                    "properties": {
                        "leader": {"type": "string"},
                        "title": {"type": "string"},
                        "source_count": {"type": "number"}
                    },
                    "required": ["leader", "title"]
                },
                contents={"text": True}
            )

            print(results)
        - lang: javascript
          label: Deep search with query variations
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.search('Who is the CEO of OpenAI?', {
                additionalQueries: [
                    'OpenAI CEO current',
                    'OpenAI leadership official source'
                ],
                type: 'deep',
                systemPrompt: 'Prefer official sources and avoid duplicate results',
                outputSchema: {
                    type: 'object',
                    properties: {
                        leader: { type: 'string' },
                        title: { type: 'string' },
                        sourceCount: { type: 'number' }
                    },
                    required: ['leader', 'title']
                },
                contents: {
                    text: true
                }
            });

            console.log(results);
        - lang: bash
          label: Streaming synthesized output
          source: |-
            curl --no-buffer --request POST \
              --url https://api.exa.ai/search \
              --header 'x-api-key: <token>' \
              --header 'Content-Type: application/json' \
              --data '{
                "query": "Summarize the latest AI chip launches",
                "type": "fast",
                "stream": true,
                "outputSchema": {
                  "type": "text",
                  "description": "A short grounded summary in 3 bullets"
                }
              }'
        - lang: bash
          label: Instant search (lowest latency)
          source: |-
            curl --request POST \
              --url https://api.exa.ai/search \
              --header 'x-api-key: <token>' \
              --header 'Content-Type: application/json' \
              --data '{
                "query": "What is the capital of France?",
                "type": "instant",
                "numResults": 10,
                "contents": {
                  "highlights": true
                }
              }'
        - lang: python
          label: Instant search (lowest latency)
          source: |-
            # pip install exa-py
            from exa_py import Exa
            exa = Exa(api_key='YOUR_EXA_API_KEY')

            results = exa.search(
                "What is the capital of France?",
                type="instant",
                num_results=10,
                contents={"highlights": True}
            )

            print(results)
        - lang: javascript
          label: Instant search (lowest latency)
          source: |-
            // npm install exa-js
            import Exa from 'exa-js';
            const exa = new Exa('YOUR_EXA_API_KEY');

            const results = await exa.search(
                'What is the capital of France?',
                {
                    type: 'instant',
                    numResults: 10,
                    contents: { highlights: true }
                }
            );

            console.log(results);
components:
  schemas:
    SearchRequest:
      type: object
      properties:
        includeDomains:
          anyOf:
            - maxItems: 1200
              type: array
              items:
                type: string
              description: >-
                List of domains to include in the search. If specified, results
                will only come from these domains.
              example:
                - arxiv.org
                - paperswithcode.com
            - type: 'null'
        excludeDomains:
          anyOf:
            - maxItems: 1200
              type: array
              items:
                type: string
              description: >-
                List of domains to exclude from search results. If specified, no
                results will be returned from these domains.
            - type: 'null'
        startCrawlDate:
          anyOf:
            - type: string
              description: >-
                Crawl date refers to the date that Exa discovered a link.
                Results will include links that were crawled after this date.
                Must be specified in ISO 8601 format.
              example: '2023-01-01T00:00:00.000Z'
              format: date-time
            - type: 'null'
        endCrawlDate:
          anyOf:
            - type: string
              description: >-
                Crawl date refers to the date that Exa discovered a link.
                Results will include links that were crawled before this date.
                Must be specified in ISO 8601 format.
              example: '2023-12-31T00:00:00.000Z'
              format: date-time
            - type: 'null'
        startPublishedDate:
          anyOf:
            - type: string
              description: >-
                Only links with a published date after this will be returned.
                Must be specified in ISO 8601 format.
              example: '2023-01-01T00:00:00.000Z'
              format: date-time
            - type: 'null'
        endPublishedDate:
          anyOf:
            - type: string
              description: >-
                Only links with a published date before this will be returned.
                Must be specified in ISO 8601 format.
              example: '2023-12-31T00:00:00.000Z'
              format: date-time
            - type: 'null'
        numResults:
          anyOf:
            - type: integer
              minimum: 1
              maximum: 100
              description: >-
                Number of results to return. Limits vary by search type. The
                maximum public limit is 100 results. Contact sales
                (hello@exa.ai) to discuss higher limits.
              example: 10
              default: 10
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
        moderation:
          anyOf:
            - type: boolean
              description: >-
                Enable content moderation to filter unsafe content from search
                results.
              example: true
              default: false
            - type: 'null'
        contents:
          anyOf:
            - $ref: '#/components/schemas/ContentsOptions'
              description: >-
                Content options for text, highlights, summary, extras, and
                freshness controls.
            - type: 'null'
        query:
          type: string
          minLength: 1
          description: The query string for the search.
          example: Latest developments in LLM capabilities
        additionalQueries:
          anyOf:
            - minItems: 1
              maxItems: 10
              type: array
              items:
                type: string
              description: >-
                Additional query variations for deep-search variants. Only works
                with a deep-search type. When provided, these queries are used
                alongside the main query for broader results.
              example:
                - LLM advancements
                - large language model progress
            - type: 'null'
        type:
          anyOf:
            - type: string
              enum:
                - instant
                - fast
                - auto
                - deep-lite
                - deep
                - deep-reasoning
              description: >-
                The type of search. auto is the default and intelligently
                selects the best search mode, instant provides the lowest
                latency for real-time applications, fast uses lower-latency
                search models, deep-lite is lightweight synthesis, deep performs
                in-depth research with synthesis, and deep-reasoning adds more
                reasoning.
              example: auto
              default: auto
            - type: 'null'
        category:
          anyOf:
            - type: string
              enum:
                - company
                - research paper
                - news
                - personal site
                - financial report
                - people
              description: >-
                A data category to focus on. Known categories include `company`,
                `research paper`, `news`, `personal site`, `financial report`,
                and `people`. Other strings are accepted and used as category
                hints for search. The `people` and `company` categories have
                improved quality for finding people profiles and company pages.
                Note: The `company` and `people` categories only support a
                limited set of filters. The following parameters are NOT
                supported for these categories: `startPublishedDate`,
                `endPublishedDate`, `startCrawlDate`, `endCrawlDate`,
                `excludeDomains`. For `people` category, `includeDomains` only
                accepts supported profile domains. Using unsupported parameters
                will result in a 400 error.
              example: research paper
            - type: 'null'
        userLocation:
          anyOf:
            - type: string
              description: The two-letter ISO country code of the user, e.g. US.
              example: US
            - type: 'null'
        compliance:
          anyOf:
            - type: string
              enum:
                - hipaa
              description: >-
                Enterprise-only compliance mode. Set to `hipaa` to require
                HIPAA-safe processing. Requests fail closed or restrict features
                when the requested behavior requires non-HIPAA-safe processors.
              example: hipaa
            - type: 'null'
        outputSchema:
          anyOf:
            - oneOf:
                - type: object
                  properties:
                    type:
                      type: string
                      const: text
                    description:
                      type: string
                  required:
                    - type
                - type: object
                  properties:
                    type:
                      type: string
                      const: object
                    description:
                      type: string
                    properties:
                      type: object
                      propertyNames:
                        type: string
                      additionalProperties:
                        $ref: '#/components/schemas/JsonValue'
                    required:
                      type: array
                      items:
                        type: string
                    additionalProperties:
                      type: boolean
                  required:
                    - type
                  additionalProperties:
                    $ref: '#/components/schemas/JsonValue'
              description: >-
                JSON schema for synthesized output. Supported root types are
                "text" and "object". When provided, the response includes an
                output object whose content matches this schema. Works with
                every search type and adds about 2 seconds of synthesis latency
                on top of the selected search type.
              type: object
            - type: 'null'
        systemPrompt:
          anyOf:
            - type: string
              description: >-
                Additional instructions that guide generated output or agent
                behavior. Use this for source preferences, novelty constraints,
                duplication constraints, or other behavior guidance.
              example: Prefer official sources and avoid duplicate results.
            - type: 'null'
        stream:
          anyOf:
            - type: boolean
              description: >-
                Requests server-sent events for synthesized output streaming.
                Streaming is currently used only when outputSchema is provided;
                otherwise the endpoint returns the normal JSON search response.
              default: false
            - type: 'null'
      required:
        - query
    SearchResponse:
      oneOf:
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            results:
              type: array
              items:
                $ref: '#/components/schemas/SearchResultOutput'
              description: >-
                A list of search results containing title, URL, published date,
                and author.
            resolvedSearchType:
              description: >-
                Deprecated legacy field. Current production responses may return
                an empty string; clients should not branch on this value.
              example: ''
              deprecated: true
              type: string
            context:
              type: string
              description: >-
                Deprecated. Combined context string from search results. Use
                highlights or text instead.
              deprecated: true
            costDollars:
              $ref: '#/components/schemas/CostDollarsOutput'
            output:
              $ref: '#/components/schemas/SearchSynthesisOutputOutput'
          required:
            - results
            - output
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            results:
              type: array
              items:
                $ref: '#/components/schemas/SearchResultOutput'
              description: >-
                A list of search results containing title, URL, published date,
                and author.
            resolvedSearchType:
              description: >-
                Deprecated legacy field. Current production responses may return
                an empty string; clients should not branch on this value.
              example: ''
              deprecated: true
              type: string
            context:
              type: string
              description: >-
                Deprecated. Combined context string from search results. Use
                highlights or text instead.
              deprecated: true
            costDollars:
              $ref: '#/components/schemas/CostDollarsOutput'
          required:
            - results
          additionalProperties: false
    SearchStreamChunk:
      oneOf:
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: text-delta
            delta:
              type: string
            choices:
              type: array
              items:
                type: object
                properties:
                  index:
                    type: integer
                    minimum: 0
                  delta:
                    type: object
                    properties:
                      role:
                        type: string
                        const: assistant
                      content:
                        type: string
                      citations:
                        type: array
                        items:
                          type: object
                          properties:
                            url:
                              type: string
                              format: uri
                              description: Source URL.
                            title:
                              type: string
                              description: Source title.
                            id:
                              type: string
                          required:
                            - url
                            - title
                            - id
                          additionalProperties: false
                    additionalProperties: false
                  finish_reason:
                    oneOf:
                      - type: string
                        const: stop
                      - type: 'null'
                required:
                  - index
                  - delta
                  - finish_reason
                additionalProperties: false
          required:
            - type
            - delta
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: grounding
            grounding:
              type: array
              items:
                type: object
                properties:
                  field:
                    type: string
                    description: >-
                      Field path in output.content, for example content or
                      companies[0].funding.
                  citations:
                    type: array
                    items:
                      type: object
                      properties:
                        url:
                          type: string
                          format: uri
                          description: Source URL.
                        title:
                          type: string
                          description: Source title.
                      required:
                        - url
                        - title
                      additionalProperties: false
                    description: Sources supporting this output field.
                  confidence:
                    type: string
                    enum:
                      - low
                      - medium
                      - high
                    description: Model-reported reliability for this field.
                required:
                  - field
                  - citations
                  - confidence
                additionalProperties: false
              description: Field-level grounding for synthesized output.
            citations:
              type: array
              items:
                type: object
                properties:
                  url:
                    type: string
                    format: uri
                    description: Source URL.
                  title:
                    type: string
                    description: Source title.
                  id:
                    type: string
                required:
                  - url
                  - title
                  - id
                additionalProperties: false
            choices:
              type: array
              items:
                type: object
                properties:
                  index:
                    type: integer
                    minimum: 0
                  delta:
                    type: object
                    properties:
                      role:
                        type: string
                        const: assistant
                      content:
                        type: string
                      citations:
                        type: array
                        items:
                          type: object
                          properties:
                            url:
                              type: string
                              format: uri
                              description: Source URL.
                            title:
                              type: string
                              description: Source title.
                            id:
                              type: string
                          required:
                            - url
                            - title
                            - id
                          additionalProperties: false
                    additionalProperties: false
                  finish_reason:
                    oneOf:
                      - type: string
                        const: stop
                      - type: 'null'
                required:
                  - index
                  - delta
                  - finish_reason
                additionalProperties: false
          required:
            - type
            - grounding
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: results
            results:
              type: array
              items:
                $ref: '#/components/schemas/SearchResultOutput'
          required:
            - type
            - results
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: stream-reset
            streamReset:
              type: boolean
              const: true
          required:
            - type
            - streamReset
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: done
            output:
              anyOf:
                - $ref: '#/components/schemas/SearchSynthesisOutputOutput'
                - type: 'null'
            searchTime:
              type: number
            costDollars:
              $ref: '#/components/schemas/CostDollarsOutput'
            choices:
              type: array
              items:
                type: object
                properties:
                  index:
                    type: integer
                    minimum: 0
                  delta:
                    type: object
                    properties:
                      role:
                        type: string
                        const: assistant
                      content:
                        type: string
                      citations:
                        type: array
                        items:
                          type: object
                          properties:
                            url:
                              type: string
                              format: uri
                              description: Source URL.
                            title:
                              type: string
                              description: Source title.
                            id:
                              type: string
                          required:
                            - url
                            - title
                            - id
                          additionalProperties: false
                    additionalProperties: false
                  finish_reason:
                    oneOf:
                      - type: string
                        const: stop
                      - type: 'null'
                required:
                  - index
                  - delta
                  - finish_reason
                additionalProperties: false
          required:
            - type
            - output
            - searchTime
          additionalProperties: false
        - type: object
          properties:
            requestId:
              type: string
              description: Unique identifier for the request.
              example: b5947044c4b78efa9552a7c89b306d95
            type:
              type: string
              const: error
            error:
              type: object
              properties:
                message:
                  type: string
              required:
                - message
              additionalProperties: false
          required:
            - type
            - error
          additionalProperties: false
      description: >-
        Schema for each JSON payload emitted in a `/search` server-sent event
        stream. Each event is emitted as `data: <json>` and the stream
        terminates with `data: [DONE]`, which is not represented by this JSON
        schema.
      type: object
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
    SearchSynthesisOutputOutput:
      type: object
      properties:
        content:
          description: >-
            Synthesized content. String by default, or object when outputSchema
            is provided.
          oneOf:
            - type: string
            - type: object
              propertyNames:
                type: string
              additionalProperties:
                $ref: '#/components/schemas/JsonValue'
        grounding:
          type: array
          items:
            type: object
            properties:
              field:
                type: string
                description: >-
                  Field path in output.content, for example content or
                  companies[0].funding.
              citations:
                type: array
                items:
                  type: object
                  properties:
                    url:
                      type: string
                      format: uri
                      description: Source URL.
                    title:
                      type: string
                      description: Source title.
                  required:
                    - url
                    - title
                  additionalProperties: false
                description: Sources supporting this output field.
              confidence:
                type: string
                enum:
                  - low
                  - medium
                  - high
                description: Model-reported reliability for this field.
            required:
              - field
              - citations
              - confidence
            additionalProperties: false
          description: Field-level grounding for synthesized output.
      required:
        - content
        - grounding
      additionalProperties: false
      description: Synthesized output. Returned when outputSchema is provided.
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