// Code generated by "core generate"; DO NOT EDIT.

package parse

import (
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
)

// AstType is the [gti.Type] for [Ast]
var AstType = gti.AddType(&gti.Type{Name: "cogentcore.org/core/pi/parse.Ast", IDName: "ast", Doc: "Ast is a node in the abstract syntax tree generated by the parsing step\nthe name of the node (from ki.Node) is the type of the element\n(e.g., expr, stmt, etc)\nThese nodes are generated by the parse.Rule's by matching tokens", Embeds: []gti.Field{{Name: "Node"}}, Fields: []gti.Field{{Name: "TokReg", Doc: "region in source lexical tokens corresponding to this Ast node -- Ch = index in lex lines"}, {Name: "SrcReg", Doc: "region in source file corresponding to this Ast node"}, {Name: "Src", Doc: "source code corresponding to this Ast node"}, {Name: "Syms", Doc: "stack of symbols created for this node"}}, Instance: &Ast{}})

// NewAst adds a new [Ast] with the given name to the given parent:
// Ast is a node in the abstract syntax tree generated by the parsing step
// the name of the node (from ki.Node) is the type of the element
// (e.g., expr, stmt, etc)
// These nodes are generated by the parse.Rule's by matching tokens
func NewAst(par ki.Ki, name ...string) *Ast {
	return par.NewChild(AstType, name...).(*Ast)
}

// KiType returns the [*gti.Type] of [Ast]
func (t *Ast) KiType() *gti.Type { return AstType }

// New returns a new [*Ast] value
func (t *Ast) New() ki.Ki { return &Ast{} }

// RuleType is the [gti.Type] for [Rule]
var RuleType = gti.AddType(&gti.Type{Name: "cogentcore.org/core/pi/parse.Rule", IDName: "rule", Doc: "The first step is matching which searches in order for matches within the\nchildren of parent nodes, and for explicit rule nodes, it looks first\nthrough all the explicit tokens in the rule.  If there are no explicit tokens\nthen matching defers to ONLY the first node listed by default -- you can\nadd a @ prefix to indicate a rule that is also essential to match.\n\nAfter a rule matches, it then proceeds through the rules narrowing the scope\nand calling the sub-nodes..", Embeds: []gti.Field{{Name: "Node"}}, Fields: []gti.Field{{Name: "Off", Doc: "disable this rule -- useful for testing and exploration"}, {Name: "Desc", Doc: "description / comments about this rule"}, {Name: "Rule", Doc: "the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names or symbols). For keywords use 'key:keyword'.  All tokens are matched at the same nesting depth as the start of the scope of this rule, unless they have a +D relative depth value differential before the token.  Use @ prefix for a sub-rule to require that rule to match -- by default explicit tokens are used if available, and then only the first sub-rule failing that.  Use ! by itself to define start of an exclusionary rule -- doesn't match when those rule elements DO match.  Use : prefix for a special group node that matches a single token at start of scope, and then defers to the child rules to perform full match -- this is used for FirstTokMap when there are multiple versions of a given keyword rule.  Use - prefix for tokens anchored by the end (next token) instead of the previous one -- typically just for token prior to 'EOS' but also a block of tokens that need to go backward in the middle of a sequence to avoid ambiguity can be marked with -"}, {Name: "StackMatch", Doc: "if present, this rule only fires if stack has this on it"}, {Name: "Ast", Doc: "what action should be take for this node when it matches"}, {Name: "Acts", Doc: "actions to perform based on parsed Ast tree data, when this rule is done executing"}, {Name: "OptTokMap", Doc: "for group-level rules having lots of children and lots of recursiveness, and also of high-frequency, when we first encounter such a rule, make a map of all the tokens in the entire scope, and use that for a first-pass rejection on matching tokens"}, {Name: "FirstTokMap", Doc: "for group-level rules with a number of rules that match based on first tokens / keywords, build map to directly go to that rule -- must also organize all of these rules sequentially from the start -- if no match, goes directly to first non-lookup case"}, {Name: "Rules", Doc: "rule elements compiled from Rule string"}, {Name: "Order", Doc: "strategic matching order for matching the rules"}, {Name: "FiTokMap", Doc: "map from first tokens / keywords to rules for FirstTokMap case"}, {Name: "FiTokElseIdx", Doc: "for FirstTokMap, the start of the else cases not covered by the map"}, {Name: "ExclKeyIdx", Doc: "exclusionary key index -- this is the token in Rules that we need to exclude matches for using ExclFwd and ExclRev rules"}, {Name: "ExclFwd", Doc: "exclusionary forward-search rule elements compiled from Rule string"}, {Name: "ExclRev", Doc: "exclusionary reverse-search rule elements compiled from Rule string"}}, Instance: &Rule{}})

// NewRule adds a new [Rule] with the given name to the given parent:
// The first step is matching which searches in order for matches within the
// children of parent nodes, and for explicit rule nodes, it looks first
// through all the explicit tokens in the rule.  If there are no explicit tokens
// then matching defers to ONLY the first node listed by default -- you can
// add a @ prefix to indicate a rule that is also essential to match.
//
// After a rule matches, it then proceeds through the rules narrowing the scope
// and calling the sub-nodes..
func NewRule(par ki.Ki, name ...string) *Rule {
	return par.NewChild(RuleType, name...).(*Rule)
}

// KiType returns the [*gti.Type] of [Rule]
func (t *Rule) KiType() *gti.Type { return RuleType }

// New returns a new [*Rule] value
func (t *Rule) New() ki.Ki { return &Rule{} }

// SetOff sets the [Rule.Off]:
// disable this rule -- useful for testing and exploration
func (t *Rule) SetOff(v bool) *Rule { t.Off = v; return t }

// SetDesc sets the [Rule.Desc]:
// description / comments about this rule
func (t *Rule) SetDesc(v string) *Rule { t.Desc = v; return t }

// SetRule sets the [Rule.Rule]:
// the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names or symbols). For keywords use 'key:keyword'.  All tokens are matched at the same nesting depth as the start of the scope of this rule, unless they have a +D relative depth value differential before the token.  Use @ prefix for a sub-rule to require that rule to match -- by default explicit tokens are used if available, and then only the first sub-rule failing that.  Use ! by itself to define start of an exclusionary rule -- doesn't match when those rule elements DO match.  Use : prefix for a special group node that matches a single token at start of scope, and then defers to the child rules to perform full match -- this is used for FirstTokMap when there are multiple versions of a given keyword rule.  Use - prefix for tokens anchored by the end (next token) instead of the previous one -- typically just for token prior to 'EOS' but also a block of tokens that need to go backward in the middle of a sequence to avoid ambiguity can be marked with -
func (t *Rule) SetRule(v string) *Rule { t.Rule = v; return t }

// SetStackMatch sets the [Rule.StackMatch]:
// if present, this rule only fires if stack has this on it
func (t *Rule) SetStackMatch(v string) *Rule { t.StackMatch = v; return t }

// SetAst sets the [Rule.Ast]:
// what action should be take for this node when it matches
func (t *Rule) SetAst(v AstActs) *Rule { t.Ast = v; return t }

// SetActs sets the [Rule.Acts]:
// actions to perform based on parsed Ast tree data, when this rule is done executing
func (t *Rule) SetActs(v Acts) *Rule { t.Acts = v; return t }

// SetOptTokMap sets the [Rule.OptTokMap]:
// for group-level rules having lots of children and lots of recursiveness, and also of high-frequency, when we first encounter such a rule, make a map of all the tokens in the entire scope, and use that for a first-pass rejection on matching tokens
func (t *Rule) SetOptTokMap(v bool) *Rule { t.OptTokMap = v; return t }

// SetFirstTokMap sets the [Rule.FirstTokMap]:
// for group-level rules with a number of rules that match based on first tokens / keywords, build map to directly go to that rule -- must also organize all of these rules sequentially from the start -- if no match, goes directly to first non-lookup case
func (t *Rule) SetFirstTokMap(v bool) *Rule { t.FirstTokMap = v; return t }

// SetRules sets the [Rule.Rules]:
// rule elements compiled from Rule string
func (t *Rule) SetRules(v RuleList) *Rule { t.Rules = v; return t }

// SetOrder sets the [Rule.Order]:
// strategic matching order for matching the rules
func (t *Rule) SetOrder(v ...int) *Rule { t.Order = v; return t }

// SetFiTokMap sets the [Rule.FiTokMap]:
// map from first tokens / keywords to rules for FirstTokMap case
func (t *Rule) SetFiTokMap(v map[string]*Rule) *Rule { t.FiTokMap = v; return t }

// SetFiTokElseIdx sets the [Rule.FiTokElseIdx]:
// for FirstTokMap, the start of the else cases not covered by the map
func (t *Rule) SetFiTokElseIdx(v int) *Rule { t.FiTokElseIdx = v; return t }

// SetExclKeyIdx sets the [Rule.ExclKeyIdx]:
// exclusionary key index -- this is the token in Rules that we need to exclude matches for using ExclFwd and ExclRev rules
func (t *Rule) SetExclKeyIdx(v int) *Rule { t.ExclKeyIdx = v; return t }

// SetExclFwd sets the [Rule.ExclFwd]:
// exclusionary forward-search rule elements compiled from Rule string
func (t *Rule) SetExclFwd(v RuleList) *Rule { t.ExclFwd = v; return t }

// SetExclRev sets the [Rule.ExclRev]:
// exclusionary reverse-search rule elements compiled from Rule string
func (t *Rule) SetExclRev(v RuleList) *Rule { t.ExclRev = v; return t }
