# Product

## Register

product

## Users

**Primary (real) audience:** engineers and hiring managers evaluating DatasetVC as a portfolio piece. They land in the app for a few minutes, poke at the workflow, and form a judgment about the author's engineering taste and attention to detail. The UI is itself the argument — it has to read as something a competent team would ship.

**Modeled (in-product) user:** a data / ML engineer who manages CSV datasets and needs Git-like rigor over them — immutable snapshots, primary-key-aware diffs, branching, fast-forward merges, and per-column stats over time. Their context is focused, technical work: comparing two versions, tracing lineage, confirming a dataset changed the way they expected. They value correctness and speed over hand-holding.

The job to be done: *"Show me exactly how this dataset changed, version over version, and let me trust that history is immutable."*

## Product Purpose

DatasetVC is a version-control system for CSV datasets. Every upload is a content-addressed, immutable snapshot; any two snapshots diff row-by-row by primary key (added / deleted / modified, not naive line diff); datasets branch and fast-forward-merge like Git; and per-column statistics (min/max/mean/nulls/uniques) are captured at commit time and charted over the dataset's history.

Success looks like: a visitor immediately understands what the tool does, can follow a snapshot's lineage and read a diff without a manual, and comes away convinced the system is precise and trustworthy. The interface should make the *correctness* of the underlying engine legible — the design's job is to surface rigor, not decorate it.

## Brand Personality

Calm, analytical, precise. Three words: **engineered, legible, quiet.**

The reference register is the Observable / Stripe-dashboard family: quiet surfaces, generous spacing, data-forward layouts, low chrome, monospace where data deserves it. The numbers and the diff lead; the UI recedes. Confidence is shown through restraint and exactness (aligned columns, honest empty states, no filler), not through saturated color or motion. Voice in copy is terse and technical, never chatty or salesy.

## Anti-references

- **Generic SaaS landing.** No gradient hero, no big-metric template, no per-section eyebrow kickers, no identical icon-heading-text feature-card grids. This is a tool, not a marketing page.
- **Cluttered enterprise BI.** No dense gray toolbar-heavy admin-panel feel, no everything-on-screen-at-once. Old-Tableau / legacy-admin density is the enemy of legibility.
- **Toy / playful.** No rounded-bubbly pastel styling, no emoji-forward consumer-app cuteness — it undercuts technical credibility.
- **Over-animated.** No scroll-jacking, bouncy/elastic motion, or decorative effects. Motion is functional only (state transitions, diff reveals), never theatrical.

## Design Principles

1. **Surface the rigor.** The engine is correct and immutable; the UI's job is to make that legible — content-addressed hashes, lineage, and diff math should be visible and trustworthy, not hidden behind chrome.
2. **The data leads, the UI recedes.** Quiet surfaces and generous spacing so diffs, tables, and charts are the focal point. Every pixel of chrome must earn its place.
3. **Precision over decoration.** Alignment, exact contrast, honest states, and monospace for data carry the brand. Restraint is the statement; saturated color and motion are not.
4. **Legible at a glance.** A first-time visitor should parse the workflow (snapshot → diff → branch → merge) without documentation. Hierarchy and labeling do the explaining.
5. **No filler, ever.** Empty states, loading states, and errors are designed as real moments, not afterthoughts — they're where a portfolio reviewer's eye goes.

## Accessibility & Inclusion

Target **WCAG 2.1 AA**: body text ≥ 4.5:1 and large/bold text ≥ 3:1 against its background (audit the current muted-gray-on-dark text against this), visible non-color focus states, full keyboard navigation for modals/tabs/forms, and a `prefers-reduced-motion` alternative for every animation. Diff status (added/deleted/modified) must never rely on color alone — pair it with a label, icon, or sign so it survives color-blindness and grayscale.
