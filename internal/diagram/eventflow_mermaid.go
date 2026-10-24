package diagram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportEventFlowMermaid writes the event flow ExportEventFlow draws, as
// Mermaid someone else can lay out and edit. The shape carries what the drawn
// picture carries in colour as well: a stadium for an event, a rounded box for
// an origin, a gear icon for an automation and a square box for an invariant
// refusing a command. The gear is mdi:cog, and the renderer must load its icon
// set: mmdc --iconPacks @iconify-json/mdi.
func ExportEventFlowMermaid(model *ast.Model, _ Style) ([]byte, error) {
	return exportEventFlowMermaid(model, mermaidGearIcon)
}

// ExportEventFlowMermaidNoIcons writes the same flow for a renderer that loads
// no icon set, such as GitHub's: each automation is a box in the colour of the
// gear it stands for.
func ExportEventFlowMermaidNoIcons(model *ast.Model, _ Style) ([]byte, error) {
	return exportEventFlowMermaid(model, mermaidAutomationBox)
}

// mermaidAutomationDrawing is what one Mermaid event flow writes for an
// automation, and the frontmatter and notes that drawing needs.
type mermaidAutomationDrawing struct {
	header   string
	notes    string
	shape    func(label string) string
	classDef string
}

var mermaidGearIcon = mermaidAutomationDrawing{
	header: mermaidFlowIconHeader,
	notes:  mermaidFlowIconNotes,
	shape: func(label string) string {
		return `@{ icon: "mdi:cog", label: "` + label + `", pos: "b", h: 36 }`
	},
	classDef: "    classDef automation fill:none,stroke:none,color:#16201B\n",
}

var mermaidAutomationBox = mermaidAutomationDrawing{
	header: mermaidFlowHeader,
	notes:  mermaidFlowBoxNotes,
	shape: func(label string) string {
		return `["` + label + `"]`
	},
	classDef: "    classDef automation fill:#16201B,stroke:#16201B,color:#FFFFFF\n",
}

func exportEventFlowMermaid(model *ast.Model, automation mermaidAutomationDrawing) ([]byte, error) {
	flow := CollapseEventFlow(model)

	var b strings.Builder
	b.WriteString(automation.header)
	if model != nil && model.Name != "" {
		fmt.Fprintf(&b, "%%%% %s\n", model.Name)
	}
	b.WriteString(automation.notes)

	if len(flow.Nodes) == 0 {
		return []byte(b.String()), nil
	}

	ids := mermaidFlowIDs(flow.Nodes)
	kinds := make(map[string]NodeKind, len(flow.Nodes))
	for _, node := range flow.Nodes {
		kinds[node.ID] = node.Kind
		fmt.Fprintf(&b, "    %s%s\n", ids[node.ID], mermaidFlowShape(node, automation))
	}

	b.WriteString(mermaidFlowRule)
	var refused []string
	drawn := 0
	for _, hop := range flow.Hops {
		from, known := ids[hop.From]
		to, reached := ids[hop.To]
		if !known || !reached {
			continue
		}
		refusal := kinds[hop.To] == NodeRejection
		if refusal {
			// linkStyle counts the arrows in the order they are written here.
			refused = append(refused, strconv.Itoa(drawn))
		}
		fmt.Fprintf(&b, "    %s\n", mermaidFlowHop(from, to, hop.Label, refusal))
		drawn++
	}

	b.WriteString(mermaidFlowRule)
	b.WriteString(mermaidFlowClassDefs)
	b.WriteString(automation.classDef)
	for _, group := range mermaidFlowGroups(flow.Nodes, ids) {
		fmt.Fprintf(&b, "    class %s %s\n", strings.Join(group.members, ","), group.class)
	}
	if len(refused) > 0 {
		fmt.Fprintf(&b, "    linkStyle %s stroke:%s\n", strings.Join(refused, ","), mermaidFlowAlert)
	}

	return []byte(b.String()), nil
}

func mermaidFlowShape(node Node, automation mermaidAutomationDrawing) string {
	label := mermaidFlowLabel(node)
	switch node.Kind {
	case NodeEvent:
		return `(["` + label + `"])`
	case NodeOrigin:
		return `("` + label + `")`
	case NodeAutomation:
		return automation.shape(label)
	default:
		return `["` + label + `"]`
	}
}

// mermaidFlowLabel writes inside the shape what the drawn picture writes under
// it, one line to a row: Mermaid places every node itself and leaves nowhere to
// put a caption of its own.
func mermaidFlowLabel(node Node) string {
	lines := []string{mermaidFlowText(node.Label)}
	if node.Caption != "" {
		lines = append(lines, mermaidFlowText(node.Caption))
	}

	return strings.Join(lines, "<br/>")
}

func mermaidFlowHop(from, to, label string, refused bool) string {
	arrow := "-->"
	if refused {
		arrow = "-.->"
	}
	if label == "" {
		return from + " " + arrow + " " + to
	}

	return from + " " + arrow + "|" + mermaidFlowText(label) + "| " + to
}

// mermaidFlowText writes the characters that would otherwise end a label as the
// entity codes Mermaid resolves, so a cadence keeps the quotes the DSL spells
// it with.
func mermaidFlowText(text string) string {
	return mermaidFlowEscapes.Replace(text)
}

var mermaidFlowEscapes = strings.NewReplacer(
	`"`, "#quot;",
	"<", "#60;",
	">", "#62;",
	"|", "#124;",
)

type mermaidFlowGroup struct {
	class   string
	members []string
}

// mermaidFlowGroups collects the nodes each class paints, in the order the
// classes are defined, so one line covers every node that takes a class.
func mermaidFlowGroups(nodes []Node, ids map[string]string) []mermaidFlowGroup {
	members := make(map[string][]string, len(mermaidFlowClasses))
	for _, node := range nodes {
		class := mermaidFlowClass(node)
		members[class] = append(members[class], ids[node.ID])
	}

	var groups []mermaidFlowGroup
	for _, class := range mermaidFlowClasses {
		if len(members[class]) > 0 {
			groups = append(groups, mermaidFlowGroup{class: class, members: members[class]})
		}
	}

	return groups
}

func mermaidFlowClass(node Node) string {
	switch node.Kind {
	case NodeEvent:
		if node.DeadEnd {
			return "dead"
		}
		return "event"
	case NodeAutomation:
		return "automation"
	case NodeRejection:
		return "reject"
	default:
		return "origin"
	}
}

// mermaidFlowIDs names each node for Mermaid, which takes no space or dot in an
// identifier. A name that comes through the cleaning as another node's is
// numbered, so two contexts declaring one invariant name stay two nodes.
func mermaidFlowIDs(nodes []Node) map[string]string {
	ids := make(map[string]string, len(nodes))
	taken := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		id := mermaidFlowID(node.Label)
		candidate := id
		for n := 2; ; n++ {
			if _, used := taken[candidate]; !used {
				break
			}
			candidate = id + "_" + strconv.Itoa(n)
		}
		taken[candidate] = struct{}{}
		ids[node.ID] = candidate
	}

	return ids
}

func mermaidFlowID(label string) string {
	var b strings.Builder
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	id := strings.Trim(b.String(), "_")
	if id == "" || (id[0] >= '0' && id[0] <= '9') {
		id = "n" + id
	}
	if _, reserved := mermaidReservedWords[strings.ToLower(id)]; reserved {
		id = "n" + id
	}

	return id
}

// mermaidReservedWords are the words Mermaid reads as its own where a node id
// is expected. end is the one a model reaches by itself.
var mermaidReservedWords = map[string]struct{}{
	"end":       {},
	"graph":     {},
	"flowchart": {},
	"subgraph":  {},
	"class":     {},
	"classdef":  {},
	"style":     {},
	"linkstyle": {},
	"click":     {},
	"direction": {},
}

var mermaidFlowClasses = []string{"event", "dead", "automation", "origin", "reject"}

// ELK is what keeps a flow of this shape readable; the default renderer tangles
// it. TB rather than LR: a chain drawn left to right measures a ribbon no
// document holds.
const mermaidFlowHeader = `---
config:
  layout: elk
  elk:
    mergeEdges: false
    nodePlacementStrategy: BRANDES_KOEPF
---
flowchart TB
`

// Mermaid paints an icon's caption in the edge label background, a grey box by
// default, and ignores a themeCSS rule written in the frontmatter.
const mermaidFlowIconHeader = `---
config:
  layout: elk
  themeVariables:
    edgeLabelBackground: "#FFFFFF"
  elk:
    mergeEdges: false
    nodePlacementStrategy: BRANDES_KOEPF
---
flowchart TB
`

// A comment between the frontmatter and the diagram type fails to parse, so
// every comment here follows the flowchart line. A bare %% line draws a node
// saying %%, which is why each separator carries a dash.
const mermaidFlowIconNotes = `%% Each automation is the mdi:cog icon, which the renderer must load:
%% mmdc --iconPacks @iconify-json/mdi. GitHub loads no icon set; for GitHub,
%% write --format event-flow-mermaid-no-icons.
%% The colours are the light theme, Mermaid's own theme carrying the dark one.
%% ---
`

const mermaidFlowBoxNotes = `%% Each automation is a box in the colour of the gear, so no icon set has to
%% load, which GitHub cannot do.
%% The colours are the light theme, Mermaid's own theme carrying the dark one.
%% ---
`

const mermaidFlowRule = "%% ---\n"

// mermaidFlowAlert restyles the arrow to an invariant that refuses a command.
// It is the colour the classes below paint a refusal in, which linkStyle cannot
// take a class for.
const mermaidFlowAlert = "#A33A2A"

const mermaidFlowClassDefs = `    classDef event fill:#FFE0C0,stroke:#B44E12,color:#16201B
    classDef dead fill:#FFE0C0,stroke:#A33A2A,stroke-dasharray:6 4,color:#A33A2A
    classDef origin fill:#ECEFED,stroke:#5A6861,color:#16201B
    classDef reject fill:#FFFFFF,stroke:#A33A2A,stroke-dasharray:6 4,color:#A33A2A
`
