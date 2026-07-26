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

    // model "name"
    model_definition: $ => seq(
      'model',
      $.string,
    ),

    // actor "name"
    actor_definition: $ => seq(
      'actor',
      $.string,
    ),

    // context "name" { aggregate "name" { ... } }
    context_definition: $ => seq(
      'context',
      $.string,
      '{',
      repeat($.aggregate_definition),
      '}',
    ),

    // aggregate "name" { slice "name" { ... } }
    aggregate_definition: $ => seq(
      'aggregate',
      $.string,
      '{',
      repeat($.slice_definition),
      '}',
    ),

    // slice "name" { ... }
    slice_definition: $ => seq(
      'slice',
      $.string,
      '{',
      repeat($._slice_item),
      '}',
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
    ),

    // command Name { fields { ... } }
    command_definition: $ => seq(
      'command',
      $.identifier,
      '{',
      optional($.fields_block),
      '}',
    ),

    // event Name { fields { ... } source external "..." }
    event_definition: $ => seq(
      'event',
      $.identifier,
      '{',
      repeat(choice(
        $.fields_block,
        seq('source', 'external', $.string),
      )),
      '}',
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

    // trigger Kind "Name" { actor Id reads Id }
    trigger_definition: $ => seq(
      'trigger',
      $.identifier,
      $.string,
      '{',
      repeat(choice(
        seq('actor', $.any_identifier),
        seq('reads', $.any_identifier),
      )),
      '}',
    ),

    // flow { command -> event: CmdName -> EvtName }
    flow_definition: $ => seq(
      'flow',
      '{',
      repeat($._flow_entry),
      '}',
    ),

    // command -> event: CmdName -> EvtName
    _flow_entry: $ => seq(
      'command',
      '->',
      'event',
      ':',
      $.identifier,
      '->',
      $.identifier,
    ),

    // view Name { fields { ... } subscribes [...] }
    view_definition: $ => seq(
      'view',
      $.identifier,
      '{',
      repeat(choice(
        $.fields_block,
        $.subscribes_block,
      )),
      '}',
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

    // automation Name { trigger Event command Cmd target context Name }
    automation_definition: $ => seq(
      'automation',
      $.identifier,
      '{',
      repeat(choice(
        seq('trigger', $.any_identifier),
        seq('command', $.any_identifier),
        seq('target', 'context', $.any_identifier),
      )),
      '}',
    ),

    // translation Name { external_system "Name" reads View command Cmd event Name { ... } }
    translation_definition: $ => seq(
      'translation',
      $.identifier,
      '{',
      repeat(choice(
        seq('external_system', $.string),
        seq('reads', $.any_identifier),
        seq('command', $.any_identifier),
        $.event_definition,
      )),
      '}',
    ),

    // === Lexical rules ===

    // Quoted strings: "..."
    string: $ => token(seq('"', repeat(/[^"]/), '"')),

    // Line comments: # ...
    comment: $ => token(seq('#', /.*/)),

    // PascalCase identifiers (used for DSL constructs like entity names)
    identifier: $ => /[A-Z][a-zA-Z0-9_]*/,

    // Any identifier (any case) - for field names, types, etc.
    any_identifier: $ => /[a-zA-Z_][a-zA-Z0-9_]*/,
  },
});
