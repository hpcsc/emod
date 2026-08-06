; =============================================================================
; highlights.scm — Tree-sitter highlight queries for the emod DSL
;
; Maps tree-sitter node types to standard highlight groups for syntax
; coloring in tree-sitter-capable editors (Neovim, Zed, Helix, etc.).
;
; Mirrors the TextMate grammar's scope assignments at
; editors/vscode/syntaxes/emod.tmLanguage.json
; =============================================================================

; --- Comments ---
(comment) @comment

; --- Strings ---
; Generic quoted strings; overridden for entity-name positions below.
(string) @string

; --- Keywords ---
; All DSL keywords
[
  "model"
  "actor"
  "context"
  "aggregate"
  "slice"
  "command"
  "event"
  "fields"
  "flow"
  "trigger"
  "view"
  "automation"
  "translation"
  "subscribes"
  "reads"
  "source"
  "external"
  "external_system"
  "target"
] @keyword

; --- Entity names (mapped to @function) ---

; Quoted entity names after model / actor / context / aggregate / slice / trigger
; (overrides the generic (string) @string for these positions)
(model_definition (string) @function)
(actor_definition (string) @function)
(context_definition (string) @function)
(aggregate_definition (string) @function)
(slice_definition (string) @function)
(trigger_definition (string) @function)

; Identifier entity names after command / event / view / automation / translation
(command_definition (identifier) @function)
(event_definition (identifier) @function)
(view_definition (identifier) @function)
(automation_definition (identifier) @function)
(translation_definition (identifier) @function)

; --- Field types ---
(field_line
  (any_identifier) ; field name
  (any_identifier) @type
  (#match? @type "^(string|date|timestamp|int)$"))

; --- Field modifiers ---
(field_line
  (any_identifier) ; field name
  (any_identifier) ; field type
  (any_identifier) @type.qualifier
  (#match? @type.qualifier "^(required|optional)$"))

; --- Operators ---
["->" ":"] @operator

; --- Punctuation ---
["{" "}" "[" "]" ","] @punctuation
