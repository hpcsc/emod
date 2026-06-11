; =============================================================================
; textobjects.scm — nvim-treesitter-textobjects queries for structural selection
;
; Defines @block.inner and @block.outer captures for all block types in the
; emod DSL, enabling visual selection of entire structural blocks via
; nvim-treesitter-textobjects.
;
; @block.inner  = content between braces (inside {})
; @block.outer  = entire block including keyword/name and braces
; =============================================================================

; --- @block.outer: entire block node including keyword, name, and braces ---

[
  (aggregate_definition)
  (automation_definition)
  (command_definition)
  (context_definition)
  (event_definition)
  (fields_block)
  (flow_definition)
  (slice_definition)
  (translation_definition)
  (trigger_definition)
  (view_definition)
] @block.outer

; --- @block.inner: content between braces only (inside {}) ---
; Uses @_start / @_end markers so nvim-treesitter-textobjects selects the
; range from after "{" to before "}", excluding the braces themselves.

[
  (aggregate_definition
    "{" @_start
    "}" @_end)
  (automation_definition
    "{" @_start
    "}" @_end)
  (command_definition
    "{" @_start
    "}" @_end)
  (context_definition
    "{" @_start
    "}" @_end)
  (event_definition
    "{" @_start
    "}" @_end)
  (fields_block
    "{" @_start
    "}" @_end)
  (flow_definition
    "{" @_start
    "}" @_end)
  (slice_definition
    "{" @_start
    "}" @_end)
  (translation_definition
    "{" @_start
    "}" @_end)
  (trigger_definition
    "{" @_start
    "}" @_end)
  (view_definition
    "{" @_start
    "}" @_end)
] @block.inner
