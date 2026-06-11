; =============================================================================
; indents.scm — Tree-sitter indent queries for the emod DSL
;
; Defines @indent and @dedent captures for all {...} delimited blocks (and
; the [...] delimited subscribes_block), enabling auto-indentation in Neovim
; via its tree-sitter indent provider.
; =============================================================================

; --- @indent / @dedent on braces ---

(aggregate_definition
  "{" @indent
  "}" @dedent)

(automation_definition
  "{" @indent
  "}" @dedent)

(command_definition
  "{" @indent
  "}" @dedent)

(context_definition
  "{" @indent
  "}" @dedent)

(event_definition
  "{" @indent
  "}" @dedent)

(fields_block
  "{" @indent
  "}" @dedent)

(flow_definition
  "{" @indent
  "}" @dedent)

(slice_definition
  "{" @indent
  "}" @dedent)

(translation_definition
  "{" @indent
  "}" @dedent)

(trigger_definition
  "{" @indent
  "}" @dedent)

(view_definition
  "{" @indent
  "}" @dedent)

; --- subscribes_block uses brackets instead of braces ---
(subscribes_block
  "[" @indent
  "]" @dedent)
