//go:build unit

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

// The examples are what the reference sends a reader to, so what they state is
// a claim the repository makes. TestValidate holds the separate claim that they
// pass the pipeline; these leaves assert presence only, naming no construct, so
// an example stays free to grow.
func TestExamples(t *testing.T) {
	t.Run("all_patterns.emod", func(t *testing.T) {
		t.Run("pins itself to a DSL version", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			require.True(t, model.VersionDeclared,
				"the flagship example must open with the version header a reader is meant to copy")
		})

		t.Run("describes at least one construct of every kind that accepts a description", func(t *testing.T) {
			described := describedKinds(parseExample(t, "all_patterns.emod"))

			var undescribed []string
			for _, kind := range []string{
				"model", "actor",
				"context", "aggregate", "slice", "trigger",
				"command", "event", "view", "automation", "translation",
			} {
				if !described[kind] {
					undescribed = append(undescribed, kind)
				}
			}

			require.Empty(t, undescribed,
				"no construct of these kinds carries a description, so the example does not show one there")
		})

		t.Run("names a field after the keyword an event binds its wire type with", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			const attribute = "type"
			require.Contains(t, lexer.Keywords(), attribute,
				"the collision this example demonstrates only exists while the word is a keyword")

			var named []string
			for _, field := range declaredFields(model) {
				if field.Name == attribute {
					named = append(named, field.Name)
				}
			}

			require.NotEmpty(t, named,
				"no field is named after the wire-type keyword, so the example does not show a word this batch reserved staying usable as a field name")
		})

		t.Run("binds a distinct wire type on some events and leaves another unbound", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			bound := make(map[string]bool)
			var boundEvents, unboundEvents []string
			for _, event := range declaredEvents(model) {
				if event.WireType == "" {
					unboundEvents = append(unboundEvents, event.Name)
					continue
				}
				boundEvents = append(boundEvents, event.Name)
				bound[event.WireType] = true
			}

			require.GreaterOrEqual(t, len(boundEvents), 2,
				"fewer than two events bind a wire type, so the example does not show the attribute")
			require.Len(t, bound, len(boundEvents),
				"two events bind the same wire type, so the example does not show that each is its own routing key")
			require.NotEmpty(t, unboundEvents,
				"every event binds a wire type, so the example does not show that the attribute is optional")
		})

		t.Run("delays one automation and schedules another that states no delay", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			var delayed, scheduled []string
			for _, automation := range declaredAutomations(model) {
				if automation.After != "" {
					delayed = append(delayed, automation.Name)
				}
				if automation.Schedule != "" && automation.After == "" {
					scheduled = append(scheduled, automation.Name)
				}
			}

			require.NotEmpty(t, delayed,
				"no automation states a delay, so the example does not show `after`")
			require.NotEmpty(t, scheduled,
				"no schedule-driven automation stands without a delay, so the example does not show that the two never combine")
		})

		t.Run("declares an invariant", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			var declared []string
			for _, context := range model.Contexts {
				for _, invariant := range context.Invariants {
					declared = append(declared, invariant.Name)
				}
				for _, aggregate := range context.Aggregates {
					for _, invariant := range aggregate.Invariants {
						declared = append(declared, invariant.Name)
					}
				}
			}

			require.NotEmpty(t, declared,
				"no scope declares an invariant, so the example does not show where a business rule lives")
		})

		t.Run("concludes a spec with every outcome the language offers", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			stated := make(map[string]bool)
			for _, spec := range declaredSpecs(model) {
				switch spec.Then.(type) {
				case *ast.ThenEvents:
					stated["events"] = true
				case *ast.ThenRejected:
					stated["rejected"] = true
				case *ast.ThenView:
					stated["view"] = true
				case *ast.ThenCommand:
					stated["command"] = true
				}
			}

			var unstated []string
			for _, outcome := range []string{"events", "rejected", "view", "command"} {
				if !stated[outcome] {
					unstated = append(unstated, outcome)
				}
			}

			require.Empty(t, unstated,
				"no spec concludes with these outcomes, so the example does not show them")
		})

		t.Run("states example payloads on a given, a when and a then reference", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			carried := make(map[string]bool)
			for _, spec := range declaredSpecs(model) {
				for _, given := range spec.Given {
					if len(given.Payload) > 0 {
						carried["given"] = true
					}
				}
				if spec.When != nil && len(spec.When.Payload) > 0 {
					carried["when"] = true
				}
				if outcome, ok := spec.Then.(*ast.ThenEvents); ok {
					for _, event := range outcome.Events {
						if len(event.Payload) > 0 {
							carried["then"] = true
						}
					}
				}
			}

			var bare []string
			for _, position := range []string{"given", "when", "then"} {
				if !carried[position] {
					bare = append(bare, position)
				}
			}

			require.Empty(t, bare,
				"no spec states an example payload in these positions")
		})

		t.Run("states a payload literal that is not a string", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			var unquoted []string
			for _, stated := range declaredPayloads(model) {
				if stated.Kind != ast.StringLiteral {
					unquoted = append(unquoted, stated.Name)
				}
			}

			require.NotEmpty(t, unquoted,
				"every payload value is quoted, so the example shows only one of the three literal forms")
		})

		t.Run("refuses a command on the timeline", func(t *testing.T) {
			model := parseExample(t, "all_patterns.emod")

			var refused []string
			for _, slice := range model.AllSlices() {
				for _, rejection := range slice.Rejections {
					refused = append(refused, rejection.CommandName)
				}
			}

			require.NotEmpty(t, refused,
				"no flow states a rejection entry, so the example does not show a command an invariant refuses")
		})
	})

	t.Run("specs_hotel.emod", func(t *testing.T) {
		t.Run("declares an invariant every rejection it states can name", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			var declared []string
			for _, context := range model.Contexts {
				for _, invariant := range context.Invariants {
					declared = append(declared, invariant.Name)
				}
				for _, aggregate := range context.Aggregates {
					for _, invariant := range aggregate.Invariants {
						declared = append(declared, invariant.Name)
					}
				}
			}

			require.NotEmpty(t, declared,
				"no scope declares an invariant, so the worked example does not show where a business rule lives")
		})

		t.Run("states example payloads that link a scenario by repeating a value", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			var linked []string
			for _, spec := range declaredSpecs(model) {
				if spec.When == nil {
					continue
				}
				for _, given := range spec.Given {
					for _, stated := range given.Payload {
						for _, exercised := range spec.When.Payload {
							if stated.Name == exercised.Name && stated.Value == exercised.Value {
								linked = append(linked, spec.Name)
							}
						}
					}
				}
			}

			require.NotEmpty(t, linked,
				"no spec repeats a payload value between its given and its when, so nothing states what ties the scenario together")
		})

		t.Run("refuses a command on the timeline", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			var refused []string
			for _, slice := range model.AllSlices() {
				for _, rejection := range slice.Rejections {
					refused = append(refused, rejection.CommandName)
				}
			}

			require.NotEmpty(t, refused,
				"no flow states a rejection entry, so the worked example does not show a command an invariant refuses")
		})

		t.Run("binds a wire type on an event", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			bound := make(map[string]bool)
			for _, event := range declaredEvents(model) {
				if event.WireType != "" {
					bound[event.WireType] = true
				}
			}

			require.NotEmpty(t, bound,
				"no event binds a wire type, so the worked example does not show the attribute")
		})

		t.Run("fires an automation a fixed duration after its activation event", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			var delayed []string
			for _, automation := range declaredAutomations(model) {
				if automation.After != "" && automation.OnEvent != "" {
					delayed = append(delayed, automation.Name)
				}
			}

			require.NotEmpty(t, delayed,
				"no automation states a delay on its activation event, so the worked example does not show the timer")
		})

		t.Run("reads a todo list the work it causes removes a row from", func(t *testing.T) {
			model := parseExample(t, "specs_hotel.emod")

			produced := make(map[string][]string)
			for _, slice := range model.AllSlices() {
				for _, flow := range slice.Flows {
					produced[flow.CommandName] = append(produced[flow.CommandName], flow.EventName)
				}
			}
			subscribers := make(map[string][]string)
			for _, slice := range model.AllSlices() {
				for _, view := range slice.Views {
					subscribers[view.Name] = append(subscribers[view.Name], view.Subscribes...)
				}
			}

			var closed []string
			for _, automation := range declaredAutomations(model) {
				for _, caused := range produced[automation.Command] {
					for _, subscribed := range subscribers[automation.Reads] {
						if subscribed == caused {
							closed = append(closed, automation.Name)
						}
					}
				}
			}

			require.NotEmpty(t, closed,
				"no automation reads a view subscribing to the event its own command produces, so its todo list never loses the row it acted on")
		})
	})
}

func parseExample(t *testing.T, name string) *ast.Model {
	t.Helper()
	path := filepath.Join("../../examples", name)

	source, err := os.ReadFile(path)
	require.NoError(t, err)

	tokens, lexDiags := lexer.Scan(string(source), path)
	require.Empty(t, lexDiags)

	model, parseDiags := parser.New(tokens, path).Parse()
	require.Empty(t, parseDiags)

	return model
}

func describedKinds(model *ast.Model) map[string]bool {
	kinds := make(map[string]bool)
	describe := func(kind, description string) {
		if description != "" {
			kinds[kind] = true
		}
	}

	describe("model", model.Description)
	for _, actor := range model.Actors {
		describe("actor", actor.Description)
	}
	for _, context := range model.Contexts {
		describe("context", context.Description)
		for _, aggregate := range context.Aggregates {
			describe("aggregate", aggregate.Description)
		}
		for _, slice := range context.AllSlices() {
			describe("slice", slice.Description)
			if slice.Trigger != nil {
				describe("trigger", slice.Trigger.Description)
			}
			for _, command := range slice.Commands {
				describe("command", command.Description)
			}
			for _, event := range slice.Events {
				describe("event", event.Description)
			}
			for _, view := range slice.Views {
				describe("view", view.Description)
			}
			for _, automation := range slice.Automations {
				describe("automation", automation.Description)
			}
			for _, translation := range slice.Translations {
				describe("translation", translation.Description)
				if translation.Event != nil {
					describe("event", translation.Event.Description)
				}
			}
		}
	}

	return kinds
}

func declaredEvents(model *ast.Model) []*ast.Event {
	var events []*ast.Event
	for _, slice := range model.AllSlices() {
		events = append(events, slice.Events...)
		for _, translation := range slice.Translations {
			if translation.Event != nil {
				events = append(events, translation.Event)
			}
		}
	}

	return events
}

func declaredAutomations(model *ast.Model) []*ast.Automation {
	var automations []*ast.Automation
	for _, slice := range model.AllSlices() {
		automations = append(automations, slice.Automations...)
	}

	return automations
}

func declaredSpecs(model *ast.Model) []*ast.Spec {
	var specs []*ast.Spec
	for _, slice := range model.AllSlices() {
		specs = append(specs, slice.Specs...)
	}

	return specs
}

func declaredPayloads(model *ast.Model) []*ast.PayloadField {
	var stated []*ast.PayloadField
	for _, spec := range declaredSpecs(model) {
		for _, given := range spec.Given {
			stated = append(stated, given.Payload...)
		}
		if spec.When != nil {
			stated = append(stated, spec.When.Payload...)
		}
		if outcome, ok := spec.Then.(*ast.ThenEvents); ok {
			for _, event := range outcome.Events {
				stated = append(stated, event.Payload...)
			}
		}
	}

	return stated
}

func declaredFields(model *ast.Model) []*ast.Field {
	var fields []*ast.Field
	for _, slice := range model.AllSlices() {
		fields = append(fields, slice.Fields...)
		for _, command := range slice.Commands {
			fields = append(fields, command.Fields...)
		}
		for _, view := range slice.Views {
			fields = append(fields, view.Fields...)
		}
	}
	for _, event := range declaredEvents(model) {
		fields = append(fields, event.Fields...)
	}

	return fields
}
