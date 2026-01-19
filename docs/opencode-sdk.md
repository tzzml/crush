APIs
------------------------------------------

The SDK exposes all server APIs through a type-safe client.

### Global

| Method | Description | Response |
| --- | --- | --- |
| `global.health()` | Check server health and version | `{ healthy: true, version: string }` |

#### Examples

```
const health = await client.global.health()console.log(health.data.version)
```

### App

| Method | Description | Response |
| --- | --- | --- |
| `app.log()` | Write a log entry | `boolean` |
| `app.agents()` | List all available agents | `Agent[]` |

#### Examples

```
// Write a log entryawait client.app.log({  body: {    service: "my-app",    level: "info",    message: "Operation completed",  },})
// List available agentsconst agents = await client.app.agents()
```

### Project

| Method | Description | Response |
| --- | --- | --- |
| `project.list()` | List all projects | `Project[]` |
| `project.current()` | Get current project | `Project` |

#### Examples

```
// List all projectsconst projects = await client.project.list()
// Get current projectconst currentProject = await client.project.current()
```

### Path
| Method | Description | Response |
| --- | --- | --- |
| `path.get()` | Get current path | `Path` |

#### Examples

```
// Get current path informationconst pathInfo = await client.path.get()
```

### Config

| Method | Description | Response |
| --- | --- | --- |
| `config.get()` | Get config info | `Config` |
| `config.providers()` | List providers and default models | `{ providers: Provider[], default: { [key: string]: string } }` |

#### Examples

```
const config = await client.config.get()
const { providers, default: defaults } = await client.config.providers()
```

### Sessions

| Method | Description | Notes |
| --- | --- | --- |
| `session.list()` | List sessions | Returns `Session[]` |
| `session.get({ path })` | Get session | Returns `Session` |
| `session.children({ path })` | List child sessions | Returns `Session[]` |
| `session.create({ body })` | Create session | Returns `Session` |
| `session.delete({ path })` | Delete session | Returns `boolean` |
| `session.update({ path, body })` | Update session properties | Returns `Session` |
| `session.init({ path, body })` | Analyze app and create `AGENTS.md` | Returns `boolean` |
| `session.abort({ path })` | Abort a running session | Returns `boolean` |
| `session.share({ path })` | Share session | Returns `Session` |
| `session.unshare({ path })` | Unshare session | Returns `Session` |
| `session.summarize({ path, body })` | Summarize session | Returns `boolean` |
| `session.messages({ path })` | List messages in a session | Returns `{ info: Message, parts: Part[] }[]` |
| `session.message({ path })` | Get message details | Returns `{ info: Message, parts: Part[] }` |
| `session.prompt({ path, body })` | Send prompt message | `body.noReply: true` returns UserMessage (context only). Default returns `AssistantMessage` with AI response |
| `session.command({ path, body })` | Send command to session | Returns `{ info: AssistantMessage, parts: Part[] }` |
| `session.shell({ path, body })` | Run a shell command | Returns `AssistantMessage` |
| `session.revert({ path, body })` | Revert a message | Returns `Session` |
| `session.unrevert({ path })` | Restore reverted messages | Returns `Session` |
| `postSessionByIdPermissionsByPermissionId({ path, body })` | Respond to a permission request | Returns `boolean` |

#### Examples

```
// Create and manage sessionsconst session = await client.session.create({  body: { title: "My session" },})
const sessions = await client.session.list()
// Send a prompt messageconst result = await client.session.prompt({  path: { id: session.id },  body: {    model: { providerID: "anthropic", modelID: "claude-3-5-sonnet-20241022" },    parts: [{ type: "text", text: "Hello!" }],  },})
// Inject context without triggering AI response (useful for plugins)await client.session.prompt({  path: { id: session.id },  body: {    noReply: true,    parts: [{ type: "text", text: "You are a helpful assistant." }],  },})
```

### Files

| Method | Description | Response |
| --- | --- | --- |
| `find.text({ query })` | Search for text in files | Array of match objects with `path`, `lines`, `line_number`, `absolute_offset`, `submatches` |
| `find.files({ query })` | Find files and directories by name | `string[]` (paths) |
| `find.symbols({ query })` | Find workspace symbols | `Symbol[]` |
| `file.read({ query })` | Read a file | `{ type: "raw" | "patch", content: string }` |
| `file.status({ query? })` | Get status for tracked files | `File[]` |

`find.files` supports a few optional query fields:

-   `type`: `"file"` or `"directory"`
-   `directory`: override the project root for the search
-   `limit`: max results (1–200)

#### Examples

```
// Search and read filesconst textResults = await client.find.text({  query: { pattern: "function.*opencode" },})
const files = await client.find.files({  query: { query: "*.ts", type: "file" },})
const directories = await client.find.files({  query: { query: "packages", type: "directory", limit: 20 },})
const content = await client.file.read({  query: { path: "src/index.ts" },})
```

### Auth

| Method | Description | Response |
| --- | --- | --- |
| `auth.set({ ... })` | Set authentication credentials | `boolean` |
