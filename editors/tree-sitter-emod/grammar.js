const buildDescribedBlock = ($, ...items) => seq(
  '{',
  repeat(choice($.description, ...items)),
  '}',
);

module.exports = grammar({
  name: 'emod',

  extras: $ => [
    /\s/,
    $.comment,
  ],

  word: $ => $.identifier,

  rules: {
    // === Top-level ===
    source_file: $ => seq(
      optional($.version_header),
      repeat($._declaration),
    ),

    // The separator lives inside the keyword token and the digits are
    // immediate: relax either half and `extras` skips the newline, so a bare
    // `emod` pairs up with a number on the next line to form a header.
    version_header: $ => seq(
      alias(token(seq('emod', /[ \t]+/)), 'emod'),
      alias(token.immediate(/[0-9]+/), $.integer),
    ),

    _declaration: $ => choice(
      $.model_definition,
      $.actor_definition,
      $.context_definition,
    ),

    // model "name" [{ ... }]
    model_definition: $ => seq(
      'model',
      $.string,
      optional(buildDescribedBlock($)),
    ),

    // actor "name" [{ ... }]
    actor_definition: $ => seq(
      'actor',
      $.string,
      optional(buildDescribedBlock($)),
    ),

    description: $ => seq(
      'description',
      $.string,
    ),

    invariant: $ => seq(
      'invariant',
      $.any_identifier,
      $.string,
    ),

    // context "name" [mode dcb] { aggregate "name" { ... } }
    context_definition: $ => seq(
      'context',
      $.string,
      optional($.mode_clause),
      buildDescribedBlock($, $.invariant, $.aggregate_definition, $.slice_definition),
    ),

    // mode dcb | mode aggregate | mode mixed — the value is a free identifier
    // so that `aggregate` here reads as the name of a mode, not as the
    // declaration keyword, matching what checkIdentifierLike accepts.
    mode_clause: $ => seq(
      'mode',
      $.any_identifier,
    ),

    // aggregate "name" { slice "name" { ... } }
    aggregate_definition: $ => seq(
      'aggregate',
      $.string,
      buildDescribedBlock($, $.invariant, $.slice_definition),
    ),

    // slice "name" { ... }
    slice_definition: $ => seq(
      'slice',
      $.string,
      buildDescribedBlock($, $._slice_item),
    ),

    // === Inside slice ===
    _slice_item: $ => choice(
      $.command_definition,
      $.event_definition,
      $.flow_definition,
      $.trigger_definition,
      $.view_definition,
      $.automation_definition,
      $.translation_definition,
      $.fields_block,
      $.spec_definition,
    ),

    // spec "name" { given [Evt { field: "value" }] when Cmd { field: 42 } then [Evt] | then rejected Name | then view Name | then command Name }
    spec_definition: $ => seq(
      'spec',
      $.string,
      '{',
      repeat(choice($.spec_given, $.spec_when, $.spec_then)),
      '}',
    ),

    spec_given: $ => seq(
      'given',
      $.spec_event_list,
    ),

    spec_when: $ => seq(
      'when',
      $._spec_element,
    ),

    spec_then: $ => seq(
      'then',
      choice(
        $.spec_event_list,
        seq('rejected', $.any_identifier),
        seq('view', $.any_identifier),
        seq('command', $.any_identifier),
      ),
    ),

    spec_event_list: $ => seq(
      '[',
      optional(seq(
        $._spec_element,
        repeat(seq(',', $._spec_element)),
      )),
      ']',
    ),

    // Hidden, so a reference that states a payload keeps the tree a names-only
    // one has always parsed to.
    _spec_element: $ => seq(
      $.any_identifier,
      optional($.payload),
    ),

    // { field: "value", other: 12.50, flag: true }
    payload: $ => seq(
      '{',
      repeat(seq($.payload_field, optional(','))),
      '}',
    ),

    payload_field: $ => seq(
      $.any_identifier,
      ':',
      choice($.string, $.number, $.boolean),
    ),

    // command Name { decides_on { ... } fields { ... } }
    command_definition: $ => seq(
      'command',
      $.identifier,
      buildDescribedBlock($, $.fields_block, $.decides_on_block),
    ),

    // decides_on { events [Evt, Evt] where <predicate> }
    decides_on_block: $ => seq(
      'decides_on',
      '{',
      repeat(choice($.events_list, $.where_clause)),
      '}',
    ),

    // events [Evt, Evt]
    events_list: $ => seq(
      'events',
      '[',
      optional(seq(
        $.any_identifier,
        repeat(seq(',', $.any_identifier)),
      )),
      ']',
    ),

    // where tag(a = b) and not (tag(c = d) or tag(e = "f"))
    where_clause: $ => seq(
      'where',
      $._predicate,
    ),

    _predicate: $ => choice(
      $.tag_predicate,
      $.not_predicate,
      $.logical_predicate,
      $.parenthesized_predicate,
    ),

    // `not` outranks both binary operators so that `not X and Y` groups as
    // `(not X) and Y`, which is how parseNotExpr recurses into its operand only.
    not_predicate: $ => prec(3, seq(
      'not',
      $._predicate,
    )),

    logical_predicate: $ => choice(
      prec.left(2, seq($._predicate, 'and', $._predicate)),
      prec.left(1, seq($._predicate, 'or', $._predicate)),
    ),

    parenthesized_predicate: $ => seq(
      '(',
      $._predicate,
      ')',
    ),

    // tag(key = fieldRef) — the value may also be a quoted literal
    tag_predicate: $ => seq(
      'tag',
      '(',
      $.any_identifier,
      '=',
      choice($.any_identifier, $.string),
      ')',
    ),

    // event Name { tags { ... } fields { ... } source external "..." }
    event_definition: $ => seq(
      'event',
      $.identifier,
      buildDescribedBlock(
        $,
        $.tags_block,
        $.fields_block,
        seq('source', 'external', $.string),
      ),
    ),

    // tags { key: fieldRef ... }
    tags_block: $ => seq(
      'tags',
      '{',
      repeat($.tag_entry),
      '}',
    ),

    tag_entry: $ => seq(
      $.any_identifier,
      ':',
      $.any_identifier,
    ),

    // fields { name type [modifier] ... }
    fields_block: $ => seq(
      'fields',
      '{',
      repeat($.field_line),
      '}',
    ),

    // Field line: name type [modifier]
    // prec.right resolves shift-reduce conflict: prefer shifting the optional modifier
    // rather than reducing early and treating it as the next field's name.
    field_line: $ => prec.right(seq(
      $.any_identifier,
      $.any_identifier,
      optional($.any_identifier),
    )),

    // trigger "Name" { actor Id reads Id }
    trigger_definition: $ => seq(
      'trigger',
      $.string,
      buildDescribedBlock(
        $,
        seq('actor', $.any_identifier),
        seq('reads', $.any_identifier),
      ),
    ),

    // flow { command -> event: CmdName -> EvtName | command -> rejected: CmdName -> invName }
    flow_definition: $ => seq(
      'flow',
      '{',
      repeat(choice($.flow_event_entry, $.flow_rejection_entry)),
      '}',
    ),

    // command -> event: CmdName -> EvtName
    flow_event_entry: $ => seq(
      'command',
      '->',
      'event',
      ':',
      $.identifier,
      '->',
      $.identifier,
    ),

    // command -> rejected: CmdName -> invName
    // The invariant takes any_identifier, as `invariant` and `spec_then` do:
    // the Go parser accepts any identifier there, and a grammar stricter than
    // the language red-squiggles a file emod validate accepts.
    flow_rejection_entry: $ => seq(
      'command',
      '->',
      'rejected',
      ':',
      $.identifier,
      '->',
      $.any_identifier,
    ),

    // view Name { fields { ... } subscribes [...] }
    view_definition: $ => seq(
      'view',
      $.identifier,
      buildDescribedBlock($, $.fields_block, $.subscribes_block),
    ),

    // subscribes [Event, Event]
    subscribes_block: $ => seq(
      'subscribes',
      '[',
      optional(seq(
        $.any_identifier,
        repeat(seq(',', $.any_identifier)),
      )),
      ']',
    ),

    // automation Name { on Event every "5m" reads View command Cmd target context Name }
    automation_definition: $ => seq(
      'automation',
      $.identifier,
      buildDescribedBlock(
        $,
        seq('on', $.any_identifier),
        seq('every', $.string),
        seq('reads', $.any_identifier),
        seq('command', $.any_identifier),
        seq('target', 'context', $.any_identifier),
      ),
    ),

    // translation Name { external_system "Name" reads View command Cmd event Name { ... } }
    translation_definition: $ => seq(
      'translation',
      $.identifier,
      buildDescribedBlock(
        $,
        seq('external_system', $.string),
        seq('reads', $.any_identifier),
        seq('command', $.any_identifier),
        $.event_definition,
      ),
    ),

    // === Lexical rules ===

    // Quoted strings: "..."
    string: $ => token(seq('"', repeat(/[^"]/), '"')),

    // Unsigned numbers with an optional fractional part: 42, 12.50
    number: $ => token(/\d+(\.\d+)?/),

    // true and false hold only in a payload's value slot, so a fields block
    // declaring a field named true still reads it as an identifier.
    boolean: $ => choice('true', 'false'),

    // Line comments: # ...
    comment: $ => token(seq('#', /.*/)),

    // PascalCase identifiers (used for DSL constructs like entity names)
    identifier: $ => /[A-Z][a-zA-Z0-9_]*/,

    // Any identifier (any case) - for field names, types, etc.
    any_identifier: $ => /[a-zA-Z_][a-zA-Z0-9_]*/,
  },
});
