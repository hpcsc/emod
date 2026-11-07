package ast

import (
	"fmt"
)

type SliceRecord struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Context     string `json:"context"`
	KeyElements string `json:"keyElements"`
}

func SliceRecords(model *Model) []SliceRecord {
	refs := model.SliceRefs()
	records := make([]SliceRecord, 0, len(refs))
	for _, ref := range refs {
		pattern, keyElements := classify(ref.Slice)
		records = append(records, SliceRecord{
			Name:        ref.Slice.Name,
			Pattern:     pattern,
			Context:     ref.Context.Name,
			KeyElements: keyElements,
		})
	}
	return records
}

func classify(s *Slice) (pattern string, keyElements string) {
	if len(s.Translations) > 0 {
		tr := s.Translations[0]
		evt := ""
		if tr.Event != nil {
			evt = tr.Event.Name
		}
		return "translation", joinNamed(tr.Command, evt)
	}
	if len(s.Automations) > 0 {
		auto := s.Automations[0]
		return "automation", fmt.Sprintf("%s, %s", activationOf(auto), auto.Command)
	}
	if len(s.Views) > 0 {
		return "view", s.Views[0].Name
	}
	if s.Trigger != nil && len(s.Commands) > 0 && len(s.Flows) > 0 {
		return "command", fmt.Sprintf("%s, %s", s.Flows[0].CommandName, s.Flows[0].EventName)
	}
	if s.Trigger != nil && len(s.Commands) > 0 && len(s.Events) > 0 {
		return "command", joinNamed(s.Commands[0].Name, s.Events[0].Name)
	}
	return "unknown", ""
}

func joinNamed(first, second string) string {
	if first != "" && second != "" {
		return first + ", " + second
	}
	if first != "" {
		return first
	}
	return second
}

func activationOf(auto *Automation) string {
	if auto.Schedule != "" {
		return `every "` + auto.Schedule + `"`
	}
	if auto.OnEvent != "" && auto.After != "" {
		return auto.OnEvent + ` after "` + auto.After + `"`
	}
	return auto.OnEvent
}
