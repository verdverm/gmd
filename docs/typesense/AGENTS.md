# Typesense Documentation

Original: https://github.com/typesense/typesense-website/tree/master/docs-site/content
Version: 30.2


## Update steps

```
# clean before making fresh copy
rm -rf 

# clone repo instead of crawling
git clone https://github.com/typesense/typesense-website

# copy relevant dirs
cp -R typesense-website/docs-site/content/guide guide
cp -R typesense-website/docs-site/content/30.2/api api

# cleanup
rm -rf typesense-website
```


## Extra References

OpenAPI Spec:
- https://github.com/typesense/typesense-api-spec

Pre-made Dictionaries:
- https://dl.typesense.org/data/stemming/plurals_en_v1.jsonl
- https://github.com/algolia/synonym-dictionaries


## Layout

```bash
$ tree
.
├── AGENTS.md
├── api
│   ├── analytics-query-suggestions.md
│   ├── api-clients.md
│   ├── api-errors.md
│   ├── api-keys.md
│   ├── authentication.md
│   ├── cluster-operations.md
│   ├── collection-alias.md
│   ├── collections.md
│   ├── conversational-search-rag.md
│   ├── curation.md
│   ├── documents.md
│   ├── federated-multi-search.md
│   ├── geosearch.md
│   ├── image-search.md
│   ├── joins.md
│   ├── natural-language-search.md
│   ├── README.md
│   ├── search.md
│   ├── server-configuration.md
│   ├── stemming.md
│   ├── stopwords.md
│   ├── synonyms.md
│   ├── vector-search.md
│   └── voice-search-query.md
└── guide
    ├── ab-testing.md
    ├── ai-agents-typesense.md
    ├── angular-search-bar.md
    ├── astro-search-bar.md
    ├── backups.md
    ├── boolean-tag-search.md
    ├── building-a-search-application.md
    ├── configure-typesense.md
    ├── data-access-control.md
    ├── docker-swarm-high-availability.md
    ├── docsearch.md
    ├── dynamodb-full-text-search.md
    ├── faqs.md
    ├── firebase-full-text-search.md
    ├── gin-search-api.md
    ├── github-actions.md
    ├── high-availability.md
    ├── install-typesense.md
    ├── installing-a-client.md
    ├── laravel-full-text-search.md
    ├── locale.md
    ├── magento2-search.md
    ├── migrating-from-algolia.md
    ├── mongodb-full-text-search.md
    ├── natural-language-search.md
    ├── next-js-search-bar.md
    ├── nuxt-js-search-bar.md
    ├── organizing-collections.md
    ├── personalization.md
    ├── query-suggestions.md
    ├── qwik-js-search-bar.md
    ├── ranking-and-relevance.md
    ├── react-native-search-bar.md
    ├── README.md
    ├── recommendations.md
    ├── reference-implementations
    │   ├── address-autocomplete.md
    │   ├── ai-image-search.md
    │   ├── airports-geo-search.md
    │   ├── books-search.md
    │   ├── boolean-search.md
    │   ├── ecommerce-storefront-with-next-js-and-typesense.md
    │   ├── ecommerce-storefront.md
    │   ├── federated-search.md
    │   ├── geo-search.md
    │   ├── good-reads-books-search-with-vue.md
    │   ├── good-reads-books-search-without-npm.md
    │   ├── guitar-chords-search-in-different-js-frameworks.md
    │   ├── hn-comments-semantic-hybrid-search.md
    │   ├── joins.md
    │   ├── kotlin-soccer-search.md
    │   ├── laravel-scout-integration.md
    │   ├── linux-commits-search.md
    │   ├── nextjs-app-router-ssr.md
    │   ├── nl-search-restaurants.md
    │   ├── pg-essays-conversational-search.md
    │   ├── README.md
    │   ├── recipe-search.md
    │   ├── songs-search.md
    │   ├── typeahead-spellchecker.md
    │   ├── typesense-autocomplete-js.md
    │   └── xkcd-search.md
    ├── running-in-production.md
    ├── search-analytics.md
    ├── search-delivery-network.md
    ├── search-ui-components.md
    ├── semantic-search.md
    ├── solid-js-search-bar.md
    ├── supabase-full-text-search.md
    ├── syncing-data-into-typesense.md
    ├── system-requirements.md
    ├── testcontainers.md
    ├── tips-for-filtering.md
    ├── tips-for-searching-common-types-of-data.md
    ├── typesense-cloud
    │   ├── role-based-access-control-admin-dashboard.md
    │   ├── search-delivery-network.md
    │   ├── single-sign-on.md
    │   └── team-accounts.md
    ├── typesense-js-client-tuning.md
    ├── updating-typesense.md
    ├── vanilla-js-search-bar.md
    └── wordpress-search.md

5 directories, 106 files
```
