package plir

import "sort"

func DominatorRows(fn Function) []DominatorRow {
	sets := computeDominators(fn)
	rows := make([]DominatorRow, 0, len(sets))
	for _, block := range fn.Blocks {
		doms := make([]string, 0, len(sets[block.ID]))
		for dom := range sets[block.ID] {
			doms = append(doms, dom)
		}
		sort.Strings(doms)
		rows = append(rows, DominatorRow{Block: block.ID, Dominators: doms})
	}
	return rows
}

func Dominates(fn Function, dominator string, block string) bool {
	if dominator == "" || block == "" {
		return false
	}
	sets := computeDominators(fn)
	return sets[block][dominator]
}

func computeDominators(fn Function) map[string]map[string]bool {
	blocks := map[string]BasicBlock{}
	all := map[string]bool{}
	entry := ""
	for _, block := range fn.Blocks {
		blocks[block.ID] = block
		all[block.ID] = true
		if block.Entry && entry == "" {
			entry = block.ID
		}
	}
	if entry == "" && len(fn.Blocks) > 0 {
		entry = fn.Blocks[0].ID
	}
	doms := map[string]map[string]bool{}
	for id := range blocks {
		doms[id] = cloneStringSet(all)
	}
	if entry != "" {
		doms[entry] = map[string]bool{entry: true}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range fn.Blocks {
			if block.ID == entry {
				continue
			}
			next := cloneStringSet(all)
			if len(block.Preds) == 0 {
				next = map[string]bool{}
			}
			for i, pred := range block.Preds {
				predSet, ok := doms[pred]
				if !ok {
					continue
				}
				if i == 0 {
					next = cloneStringSet(predSet)
					continue
				}
				next = intersectStringSets(next, predSet)
			}
			next[block.ID] = true
			if !equalStringSets(doms[block.ID], next) {
				doms[block.ID] = next
				changed = true
			}
		}
	}
	return doms
}

func cloneStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func intersectStringSets(left map[string]bool, right map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key := range left {
		if right[key] {
			out[key] = true
		}
	}
	return out
}

func equalStringSets(left map[string]bool, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
