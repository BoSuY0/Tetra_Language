package regions

const (
	None                = -1
	Unknown             = -2
	ParamStart          = -3
	ExplicitBorrowStart = -1000000
)

func CopyVars(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func CopyTree(src map[string]int) map[string]int {
	return CopyVars(src)
}

func MergeVars(a, b map[string]int) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			vb = None
		}
		if va == vb {
			if va != None {
				merged[k] = va
			}
			continue
		}
		merged[k] = Unknown
	}
	for k, vb := range b {
		if _, ok := a[k]; ok {
			continue
		}
		if vb != None {
			merged[k] = Unknown
		}
	}
	return merged
}

func Join(a, b int) int {
	if a == None {
		return b
	}
	if b == None {
		return a
	}
	if a == b {
		return a
	}
	return Unknown
}

func CommonFromTree(tree map[string]int) int {
	regionID := None
	for _, leafRegion := range tree {
		regionID = Join(regionID, leafRegion)
	}
	return regionID
}

func ConstructorFromTree(tree map[string]int) int {
	regionID := CommonFromTree(tree)
	if regionID == Unknown {
		return None
	}
	return regionID
}
