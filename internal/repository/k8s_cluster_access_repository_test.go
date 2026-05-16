package repository

import "testing"

func TestEffectiveTierIndex_ClusterAccessible(t *testing.T) {
	idx := EffectiveTierIndex{
		GlobalRank: 1,
		PerCluster: map[uint]int{5: 3},
	}
	if !idx.ClusterAccessible(99, 1) {
		t.Fatal("global readonly should cover cluster 99")
	}
	if !idx.ClusterAccessible(5, 3) {
		t.Fatal("per-cluster admin")
	}
	if idx.ClusterAccessible(5, 1) && idx.GlobalRank < 1 {
		t.Fatal("cluster 5 should not be readonly without grant")
	}
	idx2 := EffectiveTierIndex{PerCluster: map[uint]int{5: 2}}
	if !idx2.ClusterAccessible(5, 2) {
		t.Fatal("readonly_exec on cluster 5")
	}
	if idx2.ClusterAccessible(5, 3) {
		t.Fatal("should not have admin")
	}
}
