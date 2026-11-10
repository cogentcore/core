// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Transition is a transition of a property of an element, which a change
// of its declarations starts (see [Context.OnTransition]). It is not
// rendered: the element has its new style at once, but the transitionend
// event is dispatched when it would end, as pages wait for it.
type Transition struct {

	// Node is the element.
	Node *html.Node

	// Property is the name of the property, as for the propertyName of
	// its transitionend event.
	Property string

	// Delay is the time before the transition starts.
	Delay time.Duration

	// Duration is the time that the transition takes.
	Duration time.Duration
}

// transitionsOf returns the transitions of the given properties of the
// given element, which have changed, for its given declarations (with
// their transition, transition-property, transition-duration and
// transition-delay).
func transitionsOf(n *html.Node, decls map[string]string, changed []string) []Transition {
	items := transitionItems(decls)
	var ts []Transition
	for _, p := range changed {
		if !animatable(p) {
			continue
		}
		for _, it := range items {
			if it.duration <= 0 || !transitionMatches(it.property, p) {
				continue
			}
			ts = append(ts, Transition{Node: n, Property: p, Delay: it.delay, Duration: it.duration})
			break
		}
	}
	return ts
}

// transitionItem is a transition of the transition declarations of an
// element: a property (or all), with its duration and delay.
type transitionItem struct {
	property        string
	duration, delay time.Duration
}

// transitionItems returns the transitions of the given declarations: the
// longhands (transition-property, with the lists of the durations and
// delays repeated for the properties), or else the transition shorthand.
func transitionItems(decls map[string]string) []transitionItem {
	get := func(p string) string {
		if v, ok := decls[p]; ok {
			return v
		}
		return decls["-webkit-"+p]
	}
	var items []transitionItem
	if v := get("transition"); v != "" {
		for _, part := range splitTopLevel(v, ',') {
			it := transitionItem{property: "all"}
			times := 0
			for _, f := range strings.Fields(part) {
				if d, ok := cssTime(f); ok {
					if times == 0 {
						it.duration = d
					} else {
						it.delay = d
					}
					times++
				} else if !timingKeyword(f) {
					it.property = strings.ToLower(f)
				}
			}
			items = append(items, it)
		}
	}
	props, durs, delays := get("transition-property"), get("transition-duration"), get("transition-delay")
	if props == "" && durs == "" && delays == "" {
		return items
	}
	// longhands after the shorthand (as they are declared after it, which
	// is the order of most styles)
	pl := splitTopLevel(props, ',')
	if props == "" {
		pl = []string{"all"}
		if len(items) > 0 {
			pl = nil
			for _, it := range items {
				pl = append(pl, it.property)
			}
		}
	}
	timesOf := func(v string, from func(it transitionItem) time.Duration) []time.Duration {
		var ds []time.Duration
		for _, f := range splitTopLevel(v, ',') {
			d, _ := cssTime(strings.TrimSpace(f))
			ds = append(ds, d)
		}
		if len(ds) == 0 {
			for _, it := range items {
				ds = append(ds, from(it))
			}
		}
		return ds
	}
	dl := timesOf(durs, func(it transitionItem) time.Duration { return it.duration })
	yl := timesOf(delays, func(it transitionItem) time.Duration { return it.delay })
	items = nil
	for i, p := range pl {
		it := transitionItem{property: strings.ToLower(strings.TrimSpace(p))}
		if len(dl) > 0 {
			it.duration = dl[i%len(dl)]
		}
		if len(yl) > 0 {
			it.delay = yl[i%len(yl)]
		}
		items = append(items, it)
	}
	return items
}

// transitionMatches returns whether a transition of the given property (as
// in transition-property) is of the given changed property: all, the same
// property, or a shorthand of it or one of its longhands (as margin and
// margin-top).
func transitionMatches(tp, p string) bool {
	return tp == "all" || tp == p || strings.HasPrefix(p, tp+"-") || strings.HasPrefix(tp, p+"-")
}

// animatable returns whether a change of the given property can be a
// transition (not of custom properties, or of the transitions themselves,
// or of display and content).
func animatable(p string) bool {
	return !strings.HasPrefix(p, "--") && !strings.Contains(p, "transition") && !strings.Contains(p, "animation") &&
		p != "display" && p != "content" && p != "position" && p != "will-change"
}

// timingKeyword returns whether the given word of a transition is a
// timing function or a keyword that is not a property.
func timingKeyword(f string) bool {
	f = strings.ToLower(f)
	switch f {
	case "ease", "linear", "ease-in", "ease-out", "ease-in-out", "step-start", "step-end", "normal", "allow-discrete", "initial", "inherit", "unset":
		return true
	}
	return strings.HasPrefix(f, "cubic-bezier(") || strings.HasPrefix(f, "steps(") || strings.HasPrefix(f, "linear(") || strings.HasPrefix(f, "var(")
}

// cssTime returns the duration of the given CSS time (as .3s or 200ms).
func cssTime(v string) (time.Duration, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	mult := float64(time.Second)
	switch {
	case strings.HasSuffix(v, "ms"):
		v, mult = strings.TrimSuffix(v, "ms"), float64(time.Millisecond)
	case strings.HasSuffix(v, "s"):
		v = strings.TrimSuffix(v, "s")
	default:
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(f * mult), true
}
