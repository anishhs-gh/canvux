# Canvux plugins

A plugin is **any executable named `canvux-*`** on the plugin search path. It
speaks JSON over stdin/stdout — so plugins can be written in any language, with
no dynamic linking.

## Install

```sh
mkdir -p ~/.config/canvux/plugins
cp canvux-flow ~/.config/canvux/plugins/
chmod +x ~/.config/canvux/plugins/canvux-flow
canvux plugins        # verify it's discovered
```

Search path (first match wins): `$CANVUX_PLUGIN_PATH`, then `./.canvux-plugins`,
then `~/.config/canvux/plugins`.

Inside the editor, plugin commands appear in the command palette (`:`) as
`Plugin: <name> — <command>`.

## Protocol

**Manifest** — the editor runs your executable with `--canvux-manifest`; print:

```json
{
  "name": "flow",
  "version": "1.0.0",
  "description": "generate flowcharts from text",
  "commands": [
    { "name": "generate", "description": "text DSL to flowchart",
      "prompt": "Flow (e.g. a -> b -> c)" }
  ]
}
```

A command with a `prompt` makes the editor ask the user for a string first.

**Command** — the editor runs `your-exe <command-name>` and writes a request
to stdin:

```json
{ "version": 1, "command": "generate", "args": "a -> b", "doc": { ... } }
```

`doc` is the current document (same schema as a `.canvux` file). Respond on
stdout with one of:

| Response | Effect |
|---|---|
| `{"objects": [ ... ]}` | add these objects (generators, importers) |
| `{"doc": { ... }}` | replace the whole document (transforms) |
| `{"message": "..."}` | status only (exporters that write their own files) |
| `{"error": "..."}` | report a failure to the user |

Objects use the scene schema: `kind` (rect/ellipse/line/arrow/polygon/path/
bezier/text), `p1`/`p2`, `points`, `stroke`, `fill`, `filled`, `strokeWidth`,
`opacity`, etc. Added objects get fresh IDs assigned by the editor.

## Example

[`canvux-flow`](canvux-flow) is a ~120-line Python object generator that turns
`start -> parse -> check; check -> retry; retry -> parse` into a laid-out
flowchart. Read it as a template.
