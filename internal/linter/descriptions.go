package linter

// RuleDescription returns a human-readable description of the named lint rule.
// The second return value is false when no rule with that name exists.
func RuleDescription(name string) (string, bool) {
	d, ok := ruleDescriptions[name]
	return d, ok
}

var ruleDescriptions = map[string]string{
	// The two orphan rules are emitted by the validator, not the linter; they
	// are described here so `lint explain` covers every rule name the tool
	// can print.
	"orphan-command":               `A command no flow references is never exercised: nothing in the model shows what event it produces or which pathway invokes it. Either add a flow connecting the command to the event it yields, or remove the command.`,
	"orphan-event":                 `An event nothing produces cannot occur: no flow yields it, no external source emits it, and no translation delivers it. Either add the producing flow, source, or translation, or remove the event.`,
	"state-obsession":              `Events named with generic state-change suffixes like "Updated", "Changed", or "Modified" hide the business meaning of what happened. Prefer names that describe a specific business fact (e.g. "OrderShipped" instead of "OrderUpdated").`,
	"property-sourcing":            `An event named "<AggregateName><Field>Changed" tracks a single property change rather than a meaningful business event. This pattern means the event was derived from the current state of an aggregate rather than capturing what happened in the domain. Prefer an event that describes the business reason for the change.`,
	"command-in-disguise":          `Events named with suffixes like "Initiated" describe what was requested (a command) rather than what happened (an event). Events should describe facts that have already occurred, not intentions.`,
	"command-past-tense":           `Commands use the imperative mood — they express an intent (e.g. "PlaceOrder", "CancelReservation"). A name ending in "ed" (past tense) reads as something that already happened, not something being requested.`,
	"view-naming":                  `Views should have names ending in "View" so their purpose is clear in the model. For example, name a view "AvailableRoomsView" rather than just "AvailableRooms".`,
	"left-chair":                   `A command referenced by three or more flows (automations, translations, or manual flows) creates coupling — changes to that command's shape or semantics must be coordinated across multiple pathways. Consider splitting into specialized commands per use case.`,
	"god-view":                     `A view that subscribes to five or more events is likely trying to do too much. Split into smaller, more focused views that serve specific read needs.`,
	"clickbait-event":              `An event with a single ID field (e.g. "OrderId") and no additional domain data is a "clickbait event" — it announces that something happened but carries no information about what or why. Either add domain-relevant fields or inline the identifier into the parent event.`,
	"dcb-in-aggregate-mode":        `DCB-style constructs (tags on events, decides_on on commands, slices directly under a context) should only appear in contexts using "dcb" or "mixed" mode. Using them in an "aggregate" context suggests a misunderstanding of the modeling paradigm.`,
	"aggregate-in-dcb-mode":        `Aggregate blocks contain slices and represent the traditional aggregate modeling style. They should not appear in a context declared as "dcb" mode, which expects a decision-based modeling approach.`,
	"dcb/untagged-event":           `In DCB and mixed modes, every event must declare at least one tag. Tags are required because the routing infrastructure relies on them to deliver events to the correct command handlers via decides_on predicates.`,
	"dcb/query-too-broad":          `A command with a decides_on that references more than 5 event types or lacks a where clause (predicate) may read too much data and couple the command to many event shapes. This can lead to performance issues and unnecessary retriggering. Prefer narrower queries with explicit predicates.`,
	"dcb/single-tag-everywhere":    `When all commands in a DCB or mixed-mode context use only one distinct tag key across their decides_on predicates, the tag dimension is not being used to express different routing concerns. Consider introducing additional tag keys to better capture the diversity of business decisions.`,
	"dcb/orphan-tag-key":           `A tag key declared on events but never referenced in any command's decides_on predicate is an "orphan" — it adds metadata to events without ever being used for routing. Either add a command that routes on this tag key, or remove it from the events.`,
	"automation/missing-todo-list": `An automation with no "reads" entry runs straight from its activation to a command, so the model holds no todo list: nothing in it shows what work is outstanding, and retry and idempotency have nowhere to live. An automation on a schedule leaves more unsaid still — the model does not state what the processor wakes up to act on. The entry stays optional so an automation can be sketched before the view it consults exists; project a view of pending work and name it in a "reads" entry.`,
	"spec/command-without-spec":   `A command no spec exercises has nothing stating the scenarios it must satisfy. Write a spec whose "when" names this command to describe at least one outcome of invoking it. The rule fires only when the model already states at least one spec elsewhere, so a model that has not adopted specs reports nothing.`,
	"spec/no-rejection-path":      `A command exercised by one or more specs but with no rejection scenario among them has only happy-path coverage — nothing in the model states what the command refuses. Write a spec whose "when" names this command and whose "then" is a rejection naming an invariant the command's scope declares.`,
}
