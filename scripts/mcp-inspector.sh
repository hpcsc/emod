#!/bin/bash

set -euo pipefail

inspector=@modelcontextprotocol/inspector@2.14.0
binary=${1:?usage: scripts/mcp-inspector.sh <path of the emod binary>}
root=$(cd "$(dirname "$0")/.." && pwd)
model=examples/all_patterns.emod

inspect() {
	npx --yes "$inspector" --cli "$binary" mcp --root "$root" -- "$@"
}

fail() {
	echo "FAIL $*" >&2
	exit 1
}

arguments_of() {
	case "$1" in
	emod_check | emod_slices_list | emod_fmt | emod_slices_arrange) echo "{\"path\":\"$model\"}" ;;
	emod_explain_rule) echo '{"rule":"state-obsession"}' ;;
	emod_schema) echo '{}' ;;
	emod_export) echo "{\"path\":\"$model\",\"format\":\"json\"}" ;;
	emod_glossary) echo "{\"path\":\"$model\",\"format\":\"markdown\"}" ;;
	emod_diagram) echo "{\"path\":\"$model\",\"format\":\"mermaid\"}" ;;
	*) return 1 ;;
	esac
}

tools=$(inspect --method tools/list | jq -r '.tools[].name') || fail "tools/list"
[ -n "$tools" ] || fail "tools/list gave no tool"
echo "ok   handshake and tools/list: $(echo "$tools" | wc -l | tr -d ' ') tools"

for tool in $tools; do
	arguments=$(arguments_of "$tool") || fail "tools/call $tool: the script has no arguments for this tool; add them to arguments_of"
	result=$(inspect --method tools/call --tool-name "$tool" --tool-args-json "$arguments") ||
		fail "tools/call $tool $arguments: $result"
	jq -e '.isError != true and (.content | length > 0)' >/dev/null <<<"$result" ||
		fail "tools/call $tool $arguments: $result"
	echo "ok   tools/call $tool"
done

resources=$(inspect --method resources/list | jq -r '.resources[].uri') || fail "resources/list"
[ -n "$resources" ] || fail "resources/list gave no resource"
echo "ok   resources/list: $(echo "$resources" | wc -l | tr -d ' ') resources"

for uri in $resources; do
	result=$(inspect --method resources/read --uri "$uri") || fail "resources/read $uri: $result"
	jq -e '.contents | length > 0 and (.[0].text | length > 0)' >/dev/null <<<"$result" ||
		fail "resources/read $uri gave no text"
	echo "ok   resources/read $uri"
done
