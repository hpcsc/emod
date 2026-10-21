const SEVERITY_NOUNS = [
  { severity: "error", one: "error", many: "errors" },
  { severity: "warning", one: "warning", many: "warnings" },
  { severity: "info", one: "info", many: "info" },
];

function of(diagnostic) {
  const known = SEVERITY_NOUNS.some(function(noun) { return noun.severity === diagnostic.severity; });

  return known ? diagnostic.severity : "error";
}

function mostSevere(diagnostics) {
  return SEVERITY_NOUNS.find(function(noun) {
    return diagnostics.some(function(d) { return of(d) === noun.severity; });
  }).severity;
}

function sortMostSevereFirst(diagnostics) {
  const rank = function(d) {
    return SEVERITY_NOUNS.findIndex(function(noun) { return noun.severity === of(d); });
  };
  return diagnostics.slice().sort(function(a, b) { return rank(a) - rank(b); });
}

function buildCountText(diagnostics) {
  return SEVERITY_NOUNS.map(function(noun) {
    const count = diagnostics.filter(function(d) { return of(d) === noun.severity; }).length;
    return count === 0 ? "" : count + " " + (count === 1 ? noun.one : noun.many);
  }).filter(Boolean).join(", ");
}

export const Severity = {
  of,
  mostSevere,
  sortMostSevereFirst,
  buildCountText,
};
