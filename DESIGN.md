---
name: DatasetVC
description: Git-like version control for CSV datasets — a quiet, engineered dark UI where the data leads.
colors:
  primary: "#4f46e5"
  primary-hover: "#6366f1"
  primary-link: "#818cf8"
  primary-data: "#a5b4fc"
  bg: "#030712"
  surface: "#111827"
  surface-raised: "#1f2937"
  border: "#1f2937"
  border-input: "#374151"
  ink: "#f3f4f6"
  ink-secondary: "#9ca3af"
  ink-muted: "#6b7280"
  ink-faint: "#4b5563"
  added: "#34d399"
  deleted: "#f87171"
  modified: "#fbbf24"
typography:
  display:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.01em"
  title:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 600
    lineHeight: 1.4
  body:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.4
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.4
rounded:
  sm: "4px"
  lg: "8px"
  xl: "12px"
  full: "9999px"
spacing:
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "20px"
  xl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "8px 16px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-ghost:
    textColor: "{colors.ink-secondary}"
    rounded: "{rounded.sm}"
    padding: "4px 12px"
  input:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "8px 12px"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.xl}"
    padding: "16px"
  badge:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.ink-secondary}"
    rounded: "{rounded.full}"
    padding: "2px 8px"
---

# Design System: DatasetVC

## 1. Overview

**Creative North Star: "The Commit Log"**

DatasetVC looks like the data structure it manages. A version-control system for CSV datasets is, at heart, an append-only chain of immutable, content-addressed commits — so the interface wears that on its sleeve: monospace hashes you can read, a literal timeline of lineage with connected dots, and diffs color-coded the way `git` taught a generation of engineers to read change. The aesthetic is a near-black developer canvas where structure is drawn with hairline borders, not boxes-with-shadows, and the only saturated color in the room is a single indigo signal plus the three diff hues (green/red/amber) that carry meaning.

The personality is **engineered, legible, quiet**. Density is deliberate: controls and metadata are small (`text-sm` / `text-xs`) so the data and diffs own the visual weight, but nothing is cramped — generous vertical rhythm separates each commit, each card, each section. The design's job is to make the *correctness* of the engine legible: a content hash is shown, not hidden; a root commit is labeled; a cached diff says so with its millisecond cost. Confidence comes from exactness, not decoration.

This system explicitly rejects four things, carried straight from the product strategy: the **generic SaaS landing** (no gradient hero, no big-metric template, no per-section eyebrow kickers, no identical feature-card grids); the **cluttered enterprise BI panel** (no dense gray toolbars, no everything-at-once); the **toy/playful** consumer look (no pastel, no bubbly radii, no emoji); and **over-animation** (no scroll-jacking, no bounce). Motion is functional state feedback only.

**Key Characteristics:**
- Near-black canvas (`#030712`), surfaces lifted by one tonal step, never by shadow
- Hairline `1px` borders do all structural separation
- A single indigo signal color, used structurally (actions, active states, lineage, identity)
- Monospace for every piece of data: hashes, cell values, primary keys
- `git`-native diff semantics: green add / red delete / amber modify, always paired with a `+ − ~` sign
- Small, quiet chrome so the data carries the weight

## 2. Colors

A near-black tonal stack of cool grays, lit by one indigo signal and three semantic diff hues. The palette is almost entirely neutral; color appears only where it means something.

### Primary
- **Signal Indigo** (`#4f46e5`, indigo-600): the one brand color. Primary buttons, and the base of the action vocabulary. Used structurally throughout — it is the "this matters / this is active / this is identity" color.
- **Indigo Hover** (`#6366f1`, indigo-500): the lift state for primary buttons; also the active-tab underline and the timeline commit dot.
- **Link Indigo** (`#818cf8`, indigo-400): inline links and the wordmark.
- **Data Indigo** (`#a5b4fc`, indigo-300): monospace identity — snapshot hashes, primary-key chips, branch HEAD references. The lightest indigo, reserved for content that *is* an identifier.

### Secondary — Diff Semantics
Three hues that exist only inside the diff and lineage system. Never decorative.
- **Added Green** (`#34d399`, emerald-400): added rows, the `+` marker, success/commit confirmations, the "initial" root badge.
- **Deleted Red** (`#f87171`, red-400): deleted rows, the `−` marker, all error states. Old values in a modified row use a lighter red with strike-through.
- **Modified Amber** (`#fbbf24`, amber-400): modified rows, the `~` marker, schema-mismatch warnings.

### Neutral
- **Canvas** (`#030712`, gray-950): the app body background. The bottom of the stack.
- **Surface** (`#111827`, gray-900): cards, the header bar, table headers — one tonal step up from canvas.
- **Surface Raised** (`#1f2937`, gray-800): inputs, code chips, badges, hover targets — the next step up.
- **Border** (`#1f2937`, gray-800): card borders and dividers. Same value as Surface Raised; structure reads as a hairline edge, not a fill.
- **Input Border** (`#374151`, gray-700): the slightly brighter stroke on form controls.
- **Ink** (`#f3f4f6`, gray-100): primary body and heading text.
- **Ink Secondary** (`#9ca3af`, gray-400): the quiet tier — supporting text, inactive tabs, labels, timestamps, counts, placeholders, fine print. This is the floor for any text: it clears AA (≥5.7:1) on every surface in the stack.
- **Ink Muted** (`#6b7280`, gray-500): **non-text only** — reserved for decorative/`aria-hidden` separators (the breadcrumb `/`). It fails AA as body text (3.67:1 on surface); never use it for content.
- **Ink Faint** (`#4b5563`, gray-600): **borders and dividers only** — never text. Fails contrast badly (2.35:1) and is forbidden as a text color.

### Token Layer

The neutral and brand roles above are implemented as Tailwind v4 `@theme` tokens in [app.css](frontend/src/app.css), exposed as semantic utilities. **Use the token utility, not the raw scale:** `bg-canvas` `bg-surface` `bg-raised` `border-edge` `border-edge-strong` `text-ink` `text-ink-soft`, and `bg-primary` `bg-primary-hover` `text-primary-link` `text-primary-data`. Rebranding the product is a single edit to the `--color-primary*` tokens. The diff palette (`emerald` / `red` / `amber`) and translucent status fills intentionally stay on Tailwind's raw scale — they carry git-diff semantics, not brand identity.

### Named Rules
**The Token Rule.** Brand and surface colors come from semantic tokens (`bg-surface`, `text-primary-link`), never the raw Tailwind scale (`bg-gray-900`, `text-indigo-400`). A `bg-indigo-600` in a component is drift — it is invisible to a rebrand. The raw scale is allowed only for the diff/status palette and incidental hover brightening.

**The Meaning-Only Color Rule.** Color is forbidden as decoration. Indigo means action/identity; green/red/amber mean add/delete/modify. If a color isn't carrying one of those meanings, it must be a neutral gray. A diff row is green because it was *added*, never to "look nice".

**The Muted Floor Rule.** No text ever goes below Ink Secondary (`#9ca3af`, gray-400). It is the darkest text tone allowed on any surface; `gray-500` and below are non-text (decorative separators, borders, dividers) and never carry content. This keeps every label, timestamp, and placeholder above the WCAG AA 4.5:1 line.

**The Sign-Plus-Color Rule.** Diff state is never communicated by color alone. Every added/deleted/modified element pairs its hue with a redundant `+ / − / ~` sign (and strike-through for removed values), so the diff survives color-blindness and grayscale.

## 3. Typography

**Display / Body Font:** the platform UI sans (`ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto`). No webfont is loaded — this is deliberate: a system stack is instant, native to a developer tool, and gets out of the data's way.
**Data / Mono Font:** the platform monospace (`ui-monospace, SFMono-Regular, Menlo, Consolas`). Carries every value that is *data*.

**Character:** Unadorned and functional. The contrast axis in this system is **sans vs. mono**, not two display faces — prose and labels are sans, anything that is a hash, a cell value, or a key is mono. That single switch does all the typographic signaling needed.

### Hierarchy
- **Display** (700, `1.5rem`/text-2xl, `-0.01em`): page titles — "Datasets", the dataset name. The largest type in the system; there is no hero scale above it, by design.
- **Title** (600, `0.875rem`/text-sm): section and card headers ("Compare Snapshots", "New Branch"). Semibold at body size — hierarchy by weight, not size.
- **Body** (400, `0.875rem`/text-sm, 1.5): default reading text, control labels, descriptions. Cap measure at 65–75ch.
- **Label** (500, `0.75rem`/text-xs): badges, metadata, helper lines, counts.
- **Mono** (400, `0.75rem`/text-xs): snapshot hashes, diff cell values, primary-key chips, branch HEADs. The data voice.

### Named Rules
**The Mono-Is-Data Rule.** Monospace is reserved exclusively for machine values — hashes, cell contents, keys, byte/row counts that are records. Never use mono for prose or as a stylistic flourish. If a user could not copy-paste it into code, it is sans.

**The Weight-Not-Size Rule.** Hierarchy below the page title is carried by weight (600 vs. 400) at a shared `0.875rem`, not by a sprawling size scale. Keeps the dense control surfaces calm.

## 4. Elevation

This system is **flat by default**. Depth is conveyed by *tonal layering*, not shadow: canvas → surface → surface-raised is a three-step lightness climb, and a single hairline `1px` border separates each plane. Cards, the header, inputs, and tables all sit flush; they are distinguished by their background tone and their border, never by a drop shadow.

There is exactly one exception, and it is intentional: the **auth card** carries a soft `shadow-xl` to float the sign-in form over an otherwise empty viewport. That is the only shadow in the product.

### Shadow Vocabulary
- **Auth float** (`box-shadow: 0 20px 25px -5px rgba(0,0,0,0.25), 0 8px 10px -6px rgba(0,0,0,0.25)`, Tailwind `shadow-xl`): used solely on the login/register card to anchor it on a bare page.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest and separated by tone + a 1px border. A shadow is allowed only to float a standalone element over empty space (the auth card). If you reach for a shadow to make a card "pop" inside a populated layout, the answer is a tonal step or a border instead.

**The Hairline Rule.** All structural separation is a single `1px` border in `#1f2937`. Never a thick rule, never a colored side-stripe. Dividers, card edges, table row separators: all the same quiet hairline.

## 5. Components

### Buttons
- **Shape:** gently rounded (`8px` / rounded-lg) for standard buttons; tightly rounded (`4px` / rounded-sm) for inline ghost actions.
- **Primary:** Signal Indigo `#4f46e5` background, `#f3f4f6` text, `8px 16px` padding, `text-sm` medium weight. The default commit/create/compare action.
- **Hover / Focus:** background lifts to Indigo Hover `#6366f1` over a `transition-colors`. Disabled drops to `opacity-40`.
- **Ghost:** no background; `#9ca3af` text that brightens to `#d1d5db` on a `#1f2937` hover fill. Used for low-stakes row actions ("→ diff", "View").
- **Secondary (outline):** transparent with a `#374151` border that brightens on hover; used for "Sign out".

### Chips / Badges
- **Style:** pill (`rounded-full`), `2px 8px`, `text-xs`. Default is `#1f2937` surface with `#9ca3af` text.
- **Semantic variants:** "initial" → translucent emerald (`bg-emerald-900/50` + emerald-400 text); "default" branch → translucent indigo; "schema mismatch" → translucent amber; "cached"/timing → neutral surface. Badges state status, never decorate.

### Cards / Containers
- **Corner Style:** `12px` (rounded-xl).
- **Background:** Surface `#111827`.
- **Shadow Strategy:** none — see Elevation. Flat, border-separated.
- **Border:** `1px` Border `#1f2937`. On interactive cards (the dataset tiles), the border brightens toward indigo on hover and the fill lifts one tone.
- **Internal Padding:** `16px` (p-4) standard; `20px` (p-5) for primary panels; `32px` (p-8) for the auth card.

### Inputs / Fields
- **Style:** `#1f2937` fill, `1px` `#374151` border, `8px` radius, `text-sm`, `#6b7280` placeholders.
- **Focus:** border shifts to Indigo Hover `#6366f1` with `outline:none`. No glow.
- **Disabled:** parent control drops to `opacity-40`.
- **Drop zone (Upload):** a `2px` dashed `#374151` border that turns indigo on drag-over; the only dashed border in the system, justified by the drag affordance.

### Navigation
- **Header:** full-width `#111827` bar with a bottom `1px` `#1f2937` border. Indigo-400 wordmark with the dataset glyph; breadcrumb-style `/` separators; user email in `#6b7280` with an outline "Sign out".
- **Tabs:** text-only buttons; active tab is `#fff` with a `2px` Indigo Hover bottom border; inactive tabs are `#9ca3af` brightening to `#e5e7eb` on hover. No pill, no fill.

### Snapshot Timeline (signature)
The History tab renders commits as a literal lineage: an Indigo Hover `#6366f1` dot per snapshot, joined by a thin `#1f2937` vertical connector, beside a card carrying the mono hash chip, message, row count, and timestamp. This is the clearest expression of "The Commit Log" — the version graph drawn as the version graph.

### Diff Table (signature)
The product's centerpiece. A full-width mono table where each row's background encodes its change type — `emerald-950/40` added, `red-950/40` deleted, `amber-950/30` modified — and a leading sign cell (`+ − ~`) repeats the meaning without color. Modified cells stack the old value (strike-through red) above the new (green), and only the actually-changed cells get a tint. A filter bar above toggles all/added/deleted/modified and surfaces the cached-vs-computed cost. This is where rigor becomes visible; it gets the most care.

## 6. Do's and Don'ts

### Do:
- **Do** keep the canvas near-black (`#030712`) and build depth with the three-tone stack (canvas → `#111827` → `#1f2937`) plus 1px hairline borders.
- **Do** reserve indigo for action, active state, and data identity; let green/red/amber appear only inside diff and lineage semantics.
- **Do** pair every diff state with its `+ / − / ~` sign (and strike-through for deletions) so meaning never depends on color alone — this is a WCAG AA requirement here, not a nicety.
- **Do** set type hierarchy by weight at a shared `0.875rem`; use mono only for real machine values (hashes, cells, keys).
- **Do** design empty, loading, and error states as real moments — "No snapshots on this branch yet", the spinner, the inline red alert. Reviewers' eyes land there.
- **Do** verify text contrast: body and supporting text must clear 4.5:1 on its actual surface. Audit `#6b7280` (ink-muted) and below carefully against `#111827`.

### Don't:
- **Don't** build a **generic SaaS landing**: no gradient hero, no big-number metric template, no tiny tracked uppercase eyebrow above sections, no identical icon-heading-text feature-card grid.
- **Don't** drift toward a **cluttered enterprise BI panel**: no dense gray toolbars, no everything-on-screen-at-once. The data leads; chrome recedes.
- **Don't** go **toy/playful**: no pastel fills, no bubbly oversized radii, no emoji-forward styling — it undercuts the technical credibility.
- **Don't** **over-animate**: no scroll-jacking, no bounce/elastic easing, no decorative motion. Motion is functional state feedback only, with a `prefers-reduced-motion` fallback.
- **Don't** use shadows to make cards "pop" inside a populated layout — tonal step or border instead. The only shadow in the product is the auth card float.
- **Don't** use a colored side-stripe (`border-left > 1px`) on cards, rows, or alerts. Separation is the uniform 1px hairline; status is a full tint + sign, never a stripe.
- **Don't** put indigo on prose or use mono as a stylistic flourish. Color and mono both carry meaning; spending them on decoration breaks the system.
