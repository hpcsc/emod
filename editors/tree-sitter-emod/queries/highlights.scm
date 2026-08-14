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
; Every keyword this file colours, listed by hand: nothing derives the list from
; the grammar, so a keyword the grammar defines stays uncoloured until it is
; added here. TestEditorKeywordCoverage
; (test/queries/keywords_test.go) requires the list to name every spelling
; internal/lexer defines.
[
  "emod"
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
  "on"
  "every"
  "after"
  "mode"
  "tags"
  "decides_on"
  "events"
  "where"
  "tag"
  "and"
  "or"
  "not"
  "description"
  "invariant"
  "spec"
  "given"
  "when"
  "then"
  "rejected"
  "type"
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

; --- Field lines: name, type, modifier ---
; The anchors are not redundant: unanchored, each pattern also matches every
; later identifier on the line, so one token carries up to three captures and
; the consumer's precedence rule, not this file, picks the winner.
; The (comment)* steps are not redundant either: an anchor skips only anonymous
; nodes, and a comment written mid-line parses as a named child of field_line.
(field_line
  . (any_identifier) @variable.member)

(field_line
  . (any_identifier)
  (comment)*
  . (any_identifier) @type)

(field_line
  . (any_identifier)
  (comment)*
  . (any_identifier)
  (comment)*
  . (any_identifier) @type.qualifier)

; --- Operators ---
["->" ":" "="] @operator

; --- Punctuation ---
["{" "}" "[" "]" ","] @punctuation
