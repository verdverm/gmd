# SCRATCH

Let's expand/update @.design/web-providers.md in the following ways

1. Rationale, should have a 2nd line in the table about expanding search providers
2. Provider landscape, there is a trend towards content-type: text/markdown
3. html -> markdown, we should consider alterntives to Go only, make a note, we can deepen it later
4. Category (1): search, Cloudflare has an option here now too, and while it is under their "browser run" product umbrella, we should also note the more minimal options in this section.

Let's move all the browser automation / session / input simulation to a .design/web-browser-advanced.md, for the sake of the first pass, we are more focused on fetch/crawl

5. The Implementation Phases needs major rework, starts a 7, goes to 8, just one bulk blob... separate implmeentation checklist... yeah, this needs major rework
6. Let's definitely NOT write our own html->markdown, that is a fools errand
7. ok, I see more implementation plan latter, it's a fucking mess, it needs to be organized and put in one section, at the end, not buried in the middle, with testing strategies included in relevant subsections
8. move design decisions and open questions to the beginning, noting that some need to move to the new web-browser-advanced.md