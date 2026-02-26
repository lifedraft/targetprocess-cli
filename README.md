# tp — Targetprocess CLI

A fast, no-nonsense command-line client for [Targetprocess](https://www.targetprocess.com/). Query work items, manage entities, and explore your instance — all from the terminal.

Built with LLM agents in mind. Works great with humans too.

## Install

**Homebrew:**

```bash
brew install lifedraft/tap/tp
```

**Go:**

```bash
go install github.com/lifedraft/targetprocess-cli/cmd/tp@latest
```

Or grab a binary from the [releases page](https://github.com/lifedraft/targetprocess-cli/releases).

## Setup

```bash
tp config set domain https://your-instance.tpondemand.com
tp config set token your-access-token
```

Config is stored in `~/.config/tp/config.yaml`. You can also use environment variables (`TP_DOMAIN`, `TP_TOKEN`) which take precedence over the file.

## How it works

The CLI wraps both the v1 and v2 Targetprocess APIs behind a handful of commands:

- **`tp show <id>`** — Show an entity by ID (auto-detects type).
- **`tp search <type>`** — Search for entities with filters and presets.
- **`tp create <type> <name>`** — Create a new entity.
- **`tp update <id>`** — Update an existing entity.
- **`tp comment`** — List, add, or delete comments on entities.
- **`tp query`** — The power tool. Query any entity type using TP's v2 query language with filtering, projections, and aggregations.
- **`tp inspect`** — Explore the API. List entity types, browse properties, discover what's available.
- **`tp api`** — Escape hatch. Hit any API endpoint directly.
- **`tp cheatsheet`** — Print a compact reference card with syntax and examples.
- **`tp bug-report`** — Print diagnostic info for bug reports, or open a pre-filled GitHub issue.

**Auto-resolution:** Entity types are resolved automatically — `userstory`, `UserStories`, `story`, and `us` all resolve to `UserStory`. Common command synonyms also work: `tp get` → `tp show`, `tp find` → `tp search`, `tp edit` → `tp update`. You can even skip the subcommand entirely: `tp 341079` is the same as `tp show 341079`.

## Quick examples

```bash
# Show an entity by ID
tp show 341079
tp 341079                  # shorthand — same thing

# Search with presets (entity types auto-resolve)
tp search UserStory --preset open
tp search story --preset open     # alias works too
tp search bugs --preset highPriority

# Create a story
tp create UserStory "Implement dark mode" --project-id 42

# Update an entity
tp update 12345 --name "New title" --state-id 100

# Comments
tp comment list 341079
tp comment add 341079 "Looks good, @timo"

# Power queries with v2 syntax
tp query Bug -w 'entityState.isFinal!=true' -s 'id,name,priority.name as priority'
tp query Assignable -s 'id,name,entityType.name as type,entityState.name as state' \
  -w 'teamIteration!=null'
tp query Feature -s 'id,name,userStories.count as total,userStories.where(entityState.isFinal==true).count as done'

# Raw API access
tp api GET '/api/v1/Users?take=10'
```

All commands support `--output json` for structured output.

## LLM agent support

This CLI was designed to be used by AI agents (Claude, GPT, etc.) as a tool for interacting with Targetprocess. A few things make this work well:

- **`tp cheatsheet`** outputs a compact reference with full query syntax, entity types, and examples — perfect for stuffing into a system prompt. Add `--output json` for structured data.
- **`tp inspect discover`** lets agents explore available entity types and their properties at runtime, so they don't need upfront knowledge of your TP schema.
- **Error messages are teaching moments.** Common query mistakes (like writing `is null` instead of `==null`) get caught with suggestions for the correct syntax.
- **`--dry-run`** on queries shows the URL that would be called without executing it — useful for verification steps.
- **JSON output everywhere** makes parsing straightforward.

## Uninstall

**Homebrew:**

```bash
brew uninstall tp
```

**Go (manual):**

```bash
rm "$(which tp)"
```

To also remove your config: `rm -rf ~/.config/tp`

## Found a bug?

Run `tp bug-report` to grab your environment info, or go straight to filing an issue:

```bash
tp bug-report --mode open
```

This opens a GitHub issue with your environment details already filled in — you just describe what went wrong.

## License

MIT
