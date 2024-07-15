// Code generated by 'yaegi extract cogentcore.org/core/tree'. DO NOT EDIT.

package symbols

import (
	"cogentcore.org/core/tree"
	"github.com/cogentcore/yaegi/interp"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/tree/tree"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Add":               reflect.ValueOf(interp.GenericFunc("func Add[T NodeValue](p *Plan, init func(w *T)) { //yaegi:add\n\tAddAt(p, AutoPlanName(2), init)\n}")),
		"AddAt":             reflect.ValueOf(interp.GenericFunc("func AddAt[T NodeValue](p *Plan, name string, init func(w *T)) { //yaegi:add\n\tp.Add(name, func() Node {\n\t\treturn any(New[T]()).(Node)\n\t}, func(n Node) {\n\t\tinit(any(n).(*T))\n\t})\n}")),
		"AddChild":          reflect.ValueOf(interp.GenericFunc("func AddChild[T NodeValue](parent Node, init func(w *T)) { //yaegi:add\n\tname := AutoPlanName(2) // must get here to get correct name\n\tparent.AsTree().Maker(func(p *Plan) {\n\t\tAddAt(p, name, init)\n\t})\n}")),
		"AddChildAt":        reflect.ValueOf(interp.GenericFunc("func AddChildAt[T NodeValue](parent Node, name string, init func(w *T)) { //yaegi:add\n\tparent.AsTree().Maker(func(p *Plan) {\n\t\tAddAt(p, name, init)\n\t})\n}")),
		"AddChildInit":      reflect.ValueOf(interp.GenericFunc("func AddChildInit[T NodeValue](parent Node, name string, init func(w *T)) { //yaegi:add\n\tparent.AsTree().Maker(func(p *Plan) {\n\t\tAddInit(p, name, init)\n\t})\n}")),
		"AddInit":           reflect.ValueOf(interp.GenericFunc("func AddInit[T NodeValue](p *Plan, name string, init func(w *T)) { //yaegi:add\n\tfor _, child := range p.Children {\n\t\tif child.Name == name {\n\t\t\tchild.Init = append(child.Init, func(n Node) {\n\t\t\t\tinit(any(n).(*T))\n\t\t\t})\n\t\t\treturn\n\t\t}\n\t}\n\tslog.Error(\"AddInit: child not found\", \"name\", name)\n}")),
		"AddNew":            reflect.ValueOf(interp.GenericFunc("func AddNew[T Node](p *Plan, name string, new func() T, init func(w T)) { //yaegi:add\n\tp.Add(name, func() Node {\n\t\treturn new()\n\t}, func(n Node) {\n\t\tinit(n.(T))\n\t})\n}")),
		"AutoPlanName":      reflect.ValueOf(tree.AutoPlanName),
		"Break":             reflect.ValueOf(tree.Break),
		"Continue":          reflect.ValueOf(tree.Continue),
		"EscapePathName":    reflect.ValueOf(tree.EscapePathName),
		"IndexByName":       reflect.ValueOf(tree.IndexByName),
		"IndexOf":           reflect.ValueOf(tree.IndexOf),
		"InitNode":          reflect.ValueOf(tree.InitNode),
		"IsNode":            reflect.ValueOf(tree.IsNode),
		"IsRoot":            reflect.ValueOf(tree.IsRoot),
		"Last":              reflect.ValueOf(tree.Last),
		"MoveToParent":      reflect.ValueOf(tree.MoveToParent),
		"New":               reflect.ValueOf(interp.GenericFunc("func New[T NodeValue](parent ...Node) *T { //yaegi:add\n\tn := new(T)\n\tni := any(n).(Node)\n\tInitNode(ni)\n\tif len(parent) == 0 {\n\t\tni.AsTree().SetName(ni.AsTree().NodeType().IDName)\n\t\treturn n\n\t}\n\tp := parent[0]\n\tp.AsTree().Children = append(p.AsTree().Children, ni)\n\tSetParent(ni, p)\n\treturn n\n}")),
		"NewNodeBase":       reflect.ValueOf(tree.NewNodeBase),
		"NewOfType":         reflect.ValueOf(tree.NewOfType),
		"Next":              reflect.ValueOf(tree.Next),
		"NextSibling":       reflect.ValueOf(tree.NextSibling),
		"Previous":          reflect.ValueOf(tree.Previous),
		"Root":              reflect.ValueOf(tree.Root),
		"SetParent":         reflect.ValueOf(tree.SetParent),
		"SetUniqueName":     reflect.ValueOf(tree.SetUniqueName),
		"UnescapePathName":  reflect.ValueOf(tree.UnescapePathName),
		"UnmarshalRootJSON": reflect.ValueOf(tree.UnmarshalRootJSON),
		"Update":            reflect.ValueOf(tree.Update),
		"UpdateSlice":       reflect.ValueOf(tree.UpdateSlice),

		// type definitions
		"Node":         reflect.ValueOf((*tree.Node)(nil)),
		"NodeBase":     reflect.ValueOf((*tree.NodeBase)(nil)),
		"NodeValue":    reflect.ValueOf((*tree.NodeValue)(nil)),
		"Plan":         reflect.ValueOf((*tree.Plan)(nil)),
		"PlanItem":     reflect.ValueOf((*tree.PlanItem)(nil)),
		"TypePlan":     reflect.ValueOf((*tree.TypePlan)(nil)),
		"TypePlanItem": reflect.ValueOf((*tree.TypePlanItem)(nil)),

		// interface wrapper definitions
		"_Node":      reflect.ValueOf((*_cogentcore_org_core_tree_Node)(nil)),
		"_NodeValue": reflect.ValueOf((*_cogentcore_org_core_tree_NodeValue)(nil)),
	}
}

// _cogentcore_org_core_tree_Node is an interface wrapper for Node type
type _cogentcore_org_core_tree_Node struct {
	IValue          interface{}
	WAsTree         func() *tree.NodeBase
	WCopyFieldsFrom func(from tree.Node)
	WDestroy        func()
	WInit           func()
	WNodeWalkDown   func(fun func(n tree.Node) bool)
	WOnAdd          func()
	WOnChildAdded   func(child tree.Node)
	WPlanName       func() string
}

func (W _cogentcore_org_core_tree_Node) AsTree() *tree.NodeBase        { return W.WAsTree() }
func (W _cogentcore_org_core_tree_Node) CopyFieldsFrom(from tree.Node) { W.WCopyFieldsFrom(from) }
func (W _cogentcore_org_core_tree_Node) Destroy()                      { W.WDestroy() }
func (W _cogentcore_org_core_tree_Node) Init()                         { W.WInit() }
func (W _cogentcore_org_core_tree_Node) NodeWalkDown(fun func(n tree.Node) bool) {
	W.WNodeWalkDown(fun)
}
func (W _cogentcore_org_core_tree_Node) OnAdd()                       { W.WOnAdd() }
func (W _cogentcore_org_core_tree_Node) OnChildAdded(child tree.Node) { W.WOnChildAdded(child) }
func (W _cogentcore_org_core_tree_Node) PlanName() string             { return W.WPlanName() }

// _cogentcore_org_core_tree_NodeValue is an interface wrapper for NodeValue type
type _cogentcore_org_core_tree_NodeValue struct {
	IValue     interface{}
	WNodeValue func()
}

func (W _cogentcore_org_core_tree_NodeValue) NodeValue() { W.WNodeValue() }
