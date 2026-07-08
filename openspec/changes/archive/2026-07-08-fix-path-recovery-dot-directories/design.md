## Context

`internal/sessions/sessions.go`:

```go
func RecoverPath(encodedPath, stored string) string {
	if stored != "" {
		return stored
	}
	if !strings.HasPrefix(encodedPath, "-") {
		return ""
	}
	segments := strings.Split(encodedPath[1:], "-")
	if path := walkRecover("/", segments, 0); path != "" {
		return path
	}
	return "/" + strings.Join(segments, "/")
}

func walkRecover(base string, segments []string, idx int) string {
	if idx == len(segments) {
		if _, err := os.Stat(base); err == nil {
			return base
		}
		return ""
	}
	for n := 1; idx+n <= len(segments); n++ {
		component := strings.Join(segments[idx:idx+n], "-")
		candidate := filepath.Join(base, component)
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		if result := walkRecover(candidate, segments, idx+n); result != "" {
			return result
		}
	}
	return ""
}
```

Claude Code encodes a project's real path by replacing **both** `/` and `.` with `-` (documented in this repo's own `CLAUDE.md`: "Encodes each pane's `pane_current_path` (replace `/` and `.` with `-`)"). `walkRecover` only reverses the `/`→`-` and literal-`-`-in-name cases (by joining candidate segments with `-`) — it never reconstructs a literal `.`. Real-world directory names with dots (`go.bin`, `bradfordwagner.src.zmk.config.keyball.44`, etc.) that have never had an active-pane discovery (so `stored` is empty, forcing the walk) resolve to nothing, and `RecoverPath` falls through to the blind `strings.Join(segments, "/")` fallback — producing a deeply-nested nonexistent path. Confirmed on this machine: encoded `-Users-bwagner-workspace-github-bradfordwagner-src-bradfordwagner-src-zmk-config-keyball-44` should resolve to the real, existing `/Users/bwagner/workspace/github/bradfordwagner/src/bradfordwagner.src.zmk.config.keyball.44`, but resolves instead to the nonexistent `/Users/bwagner/workspace/github/bradfordwagner/src/bradfordwagner/src/zmk/config/keyball/44`. `tmux neww -c <nonexistent path>` silently falls back to tmux's default directory, so `w`/`h`/`v` in the dashboard Sessions view opens a new `claude` window in the wrong place with no visible error.

Projects that already have a cached `project_path` in `sessions.jsonl` (from a prior active-pane discovery, which always used the real `pane_current_path` verbatim) are unaffected — `RecoverPath` returns the stored value directly and never reaches `walkRecover`.

## Goals / Non-Goals

**Goals:**
- `walkRecover` correctly reconstructs directory-name components that contain literal `.` characters, not just literal `-`
- Preserve existing behavior for directory names containing literal `-` (e.g. `tmux-claude-notify`, already handled and tested via the existing hyphen-join path)
- Keep the existing "verify via filesystem" philosophy: only accept a component interpretation that `os.Stat` confirms exists

**Non-Goals:**
- Disambiguating cases where multiple interpretations (dot vs hyphen vs separator) are all simultaneously valid on disk — pick the first filesystem-verified match, consistent with today's behavior for the hyphen-only case
- Changing the "stored path used directly" fast path or the final display-fallback join
- General performance tuning of the walk beyond what's needed to keep the added dot/hyphen combination search bounded for realistic path depths

## Decisions

### Decision: For each candidate component span, try all `-`/`.` boundary combinations, not just all-hyphen

Today, for a span of `n` segments, `walkRecover` tests exactly one candidate: all segments joined with `-`. The fix generates every way of joining those `n` segments where each of the `n-1` internal boundaries is independently `-` or `.` (2^(n-1) candidates), and `os.Stat`s each until one resolves to an existing directory.

Alternative considered: only try "all dots" and "all hyphens" as two fixed candidates (not full combinatorial) — rejected because real directory names can mix separators within one component (e.g. `foo.bar-baz`), and the existing design's guarantee is "verify via filesystem," not "guess the common case."

Alternative considered: change the encoding/decoding scheme to be reversible (e.g. escape literal dots/hyphens differently) — out of scope; the encoding is owned by Claude Code itself (`~/.claude/projects/<encoded-path>/`), not this plugin, so decoding must stay heuristic.

### Decision: Bound combinatorial growth implicitly via filesystem verification, no explicit cap

The existing algorithm is already exponential in the worst case (it tries every split point via the outer `n` loop, recursively). Adding a factor of `2^(n-1)` per span increases this further, but every candidate is pruned immediately by `os.Stat` — real filesystems have shallow, mostly-unambiguous structure, so in practice the walk terminates quickly (as it already does today). No explicit span-length cap is introduced; if this proves too slow in practice for pathological inputs, a future change can add one.

## Risks / Trade-offs

- [Risk] Combinatorial blowup for very long single-component runs (many consecutive segments with no valid `/` boundary) → Mitigation: each candidate is pruned by a cheap `os.Stat`; real project paths are shallow (a handful of segments per component), matching the existing algorithm's already-exponential-but-fine-in-practice behavior.
- [Risk] A component could exist on disk under more than one dot/hyphen interpretation (rare, but possible with contrived directory names) → Mitigation: first filesystem-verified match wins, same tie-breaking behavior the existing hyphen-only path already has for split-point ambiguity.
