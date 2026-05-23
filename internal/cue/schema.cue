// Schema for emod model definition files.
// This file defines the canonical structure that all emod exports conform to.
// It does not import external CUE packages to remain self-contained and portable.

#Comment: {
	text: string
}

#Field: {
	name:     string
	type:     string
	modifier?: string // "required" or "optional"; omitted when empty
}

#Trigger: {
	comments?: [...#Comment]
	kind:     string
	name:     string
	actor?:   string
	reads?:   string
}

#Command: {
	comments?: [...#Comment]
	name:     string
	fields?:  [...#Field]
}

#Event: {
	comments?:      [...#Comment]
	name:          string
	source?:       string
	external_name?: string
	fields?:       [...#Field]
}

#Flow: {
	comments?:    [...#Comment]
	command_name: string
	event_name:   string
}

#View: {
	comments?:   [...#Comment]
	name:       string
	fields?:    [...#Field]
	subscribes?: [...string]
}

#Automation: {
	comments?:       [...#Comment]
	name:           string
	trigger_event?:  string
	command?:        string
	target_context?: string
}

#Translation: {
	comments?:        [...#Comment]
	name:            string
	external_system?: string
	reads?:          string
	command?:         string
	event?:           #Event
}

#Slice: {
	comments?:      [...#Comment]
	name:          string
	trigger?:      #Trigger
	commands?:     [...#Command]
	events?:       [...#Event]
	fields?:       [...#Field]
	flows?:        [...#Flow]
	views?:        [...#View]
	automations?:  [...#Automation]
	translations?: [...#Translation]
}

#Aggregate: {
	comments?: [...#Comment]
	name:     string
	slices?:  [...#Slice]
}

#Context: {
	comments?:   [...#Comment]
	name:       string
	aggregates?: [...#Aggregate]
}

#Actor: {
	comments?: [...#Comment]
	name:     string
}

#Model: {
	comments?:  [...#Comment]
	name:      string
	actors?:   [...#Actor]
	contexts?: [...#Context]
}
