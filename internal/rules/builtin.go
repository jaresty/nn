package rules

import _ "embed"

//go:embed builtin.dl
var builtinDL string

// BuiltinRules returns the embedded built-in ruleset source (Datalog text).
// These encode nn check's representation-subgraph invariants as violation
// clauses (ADR-0019).
func BuiltinRules() string { return builtinDL }
