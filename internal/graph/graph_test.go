package graph

import (
	"reflect"
	"sort"
	"testing"
)

func TestGetDownstream(t *testing.T) {
	g := NewGraph()

	// A -> B (A depends on B)
	// B -> C (B depends on C)
	// D -> B (D depends on B)

	// If C changes:
	// B depends on C -> B is impacted
	// A depends on B -> A is impacted
	// D depends on B -> D is impacted

	g.AddNode("public", "A", Table, "", 0)
	g.AddNode("public", "B", Table, "", 0)
	g.AddNode("public", "C", Table, "", 0)
	g.AddNode("public", "D", Table, "", 0)

	g.AddEdge("public", "A", "public", "B", ForeignKey, "fk_a_b", "NO ACTION")
	g.AddEdge("public", "B", "public", "C", ForeignKey, "fk_b_c", "NO ACTION")
	g.AddEdge("public", "D", "public", "B", ForeignKey, "fk_d_b", "NO ACTION")

	// Test impact of C
	impacted := g.GetDownstream("public.C")
	sort.Strings(impacted)

	expected := []string{"public.A", "public.B", "public.D"}
	sort.Strings(expected)

	if !reflect.DeepEqual(impacted, expected) {
		t.Errorf("Expected impacted %v, got %v", expected, impacted)
	}

	// Test impact of B
	// A depends on B
	// D depends on B
	// C is not impacted
	impactedB := g.GetDownstream("public.B")
	sort.Strings(impactedB)

	expectedB := []string{"public.A", "public.D"}
	sort.Strings(expectedB)

	if !reflect.DeepEqual(impactedB, expectedB) {
		t.Errorf("Expected impacted for B %v, got %v", expectedB, impactedB)
	}
}
