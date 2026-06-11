; =============================================================================
; folds.scm — Tree-sitter fold queries for the emod DSL
;
; Defines @fold captures for all {...} delimited blocks (and the [...]
; delimited subscribes_block), enabling code folding in Neovim via its
; tree-sitter fold provider.
; =============================================================================

[
  (aggregate_definition)
  (automation_definition)
  (command_definition)
  (context_definition)
  (event_definition)
  (fields_block)
  (flow_definition)
  (slice_definition)
  (subscribes_block)
  (translation_definition)
  (trigger_definition)
  (view_definition)
] @fold
