# Design: Dark Glassmorphism UI + Fuzzy Suggestions

**Date:** 2026-03-13
**Status:** Approved

## Overview

Two enhancements to the golinks server:
1. Redesign the web UI with a dark glassmorphism aesthetic
2. Replace 404 responses for unmatched links with a fuzzy suggestions page

## UI Redesign

**Visual language:**
- Dark background (#0a0a0f) with subtle radial gradient blobs (purple/blue) for depth
- Main content in a frosted glass card: semi-transparent background (`rgba(255,255,255,0.05)`), `backdrop-filter: blur(20px)`, thin `1px solid rgba(255,255,255,0.1)` border, 16px border-radius
- Inputs: dark semi-transparent with light borders, blue/purple glow on focus
- Buttons: primary (Add) gets blue-to-purple gradient, Delete is muted red, Edit/Save are ghost-style glass
- Table rows: subtle hover highlights, thin separators
- Typography: system font stack, white text with gray variants for secondary
- Smooth transitions (150-200ms) on hover/focus
- Pure CSS, no external dependencies

## Fuzzy Suggestions Engine

**Matching (Go stdlib only):**
- Substring matching: query contains phrase or phrase contains query
- Edit distance: Levenshtein distance, rank by lowest distance relative to query length
- Combined scoring: substring matches get a bonus, results sorted by score, top 5 shown
- New `LinkStore.Suggest(phrase string) []ScoredLink` method

**Auto-redirect:**
- Single match with edit distance <= 2: auto-redirect after 2 seconds with "Redirecting to go/{phrase}..." and cancel link
- Otherwise: show full suggestions page with clickable glass cards

**Suggestions page:**
- Same glassmorphism dark theme as main UI
- "go/{query} not found" header
- Suggestion cards showing `go/{phrase}` and destination URL
- No matches: "No matches found" with link to home page

## Files Changed

- `links.go` -- add `ScoredLink` struct and `Suggest()` method
- `handlers.go` -- replace `http.NotFound` with suggestion/auto-redirect logic, embed new template
- `templates/index.html` -- full redesign with dark glassmorphism
- `templates/suggest.html` -- new suggestions page template
- `handlers_test.go` -- update 404 test, add suggestion/redirect tests
- `links_test.go` -- add `Suggest()` tests

## Unchanged

- All API routes and behavior
- `main.go`
- JSON storage layer
- No external dependencies added
