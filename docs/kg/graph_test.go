// Package kg holds the handover knowledge graph (docs/kg). This test keeps
// graph.json consistent with the repository so the KG cannot silently drift.
package kg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type graph struct {
	Nodes []struct {
		ID, Type, Title, Summary, Status string
		Paths, Docs, Tests               []string
	}
	Edges []struct{ From, Rel, To string }
}

var (
	types = map[string]bool{"system": true, "doc": true, "external": true, "component": true, "app": true,
		"test-suite": true, "feature": true, "flow": true, "limitation": true}
	rels = map[string]bool{"documented_by": true, "contains": true, "depends_on": true, "integrates": true,
		"implements": true, "exercises": true, "affects": true}
)

const root = "../.."

func load(t *testing.T) graph {
	raw, err := os.ReadFile("graph.json")
	if err != nil {
		t.Fatal(err)
	}
	var g graph
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGraph_NodesEdgesAndPathsAreValid(t *testing.T) {
	g := load(t)
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		if n.ID == "" || ids[n.ID] || !types[n.Type] || n.Title == "" {
			t.Errorf("bad or duplicate node %+v", n)
		}
		ids[n.ID] = true
		for _, p := range append(append(append([]string{}, n.Paths...), n.Docs...), n.Tests...) {
			if _, err := os.Stat(filepath.Join(root, p)); err != nil {
				t.Errorf("node %s: path %s does not exist", n.ID, p)
			}
		}
	}
	for _, e := range g.Edges {
		if !ids[e.From] || !ids[e.To] || !rels[e.Rel] {
			t.Errorf("bad edge %+v", e)
		}
	}
}

// Every Go package and command, and every KG page, must be in the graph.
func TestGraph_CoversPackagesAndPages(t *testing.T) {
	g := load(t)
	covered := map[string]bool{}
	for _, n := range g.Nodes {
		for _, p := range n.Paths {
			covered[p] = true
		}
	}
	for _, dir := range []string{"internal", "cmd"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if p := dir + "/" + e.Name(); e.IsDir() && !covered[p] {
				t.Errorf("%s has no component node in docs/kg/graph.json", p)
			}
		}
	}
	pages, _ := filepath.Glob("*.md")
	for _, p := range pages {
		if !covered["docs/kg/"+p] {
			t.Errorf("KG page %s has no doc node", p)
		}
	}
	for _, p := range pages {
		raw, _ := os.ReadFile(p)
		for _, link := range strings.Split(string(raw), "](")[1:] {
			target := strings.SplitN(strings.SplitN(link, ")", 2)[0], "#", 2)[0]
			if target == "" || strings.Contains(target, "://") {
				continue
			}
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s links to missing %s", p, target)
			}
		}
	}
}
