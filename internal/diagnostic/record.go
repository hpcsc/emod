package diagnostic

type Record struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func Records(entries []*Entry) []Record {
	records := make([]Record, 0, len(entries))
	for _, e := range entries {
		records = append(records, Record{
			File:     e.Filename,
			Line:     e.Line,
			Rule:     e.RuleName,
			Severity: e.Severity.String(),
			Message:  e.Message,
		})
	}
	return records
}
