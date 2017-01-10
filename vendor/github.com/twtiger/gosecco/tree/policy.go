package tree

// RawPolicy represents the raw parsed rules and macros in the order they were encountered. This can be used to generate the final Policy
type RawPolicy struct {
	RuleOrMacros []interface{}
}

// Policy represents a complete policy file. It is possible to combine more than one policy file
type Policy struct {
	DefaultPositiveAction string
	DefaultNegativeAction string
	DefaultPolicyAction   string
	ActionOnX32           string
	ActionOnAuditFailure  string
	Macros                map[string]Macro
	Rules                 []*Rule
}
