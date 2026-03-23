package normalize

import "sort"

func DecodePortBitmask(mask int, portCount int) []int {
	ports := make([]int, 0, portCount)
	for port := 1; port <= portCount; port++ {
		bit := 1 << (port - 1)
		if mask&bit != 0 {
			ports = append(ports, port)
		}
	}
	return ports
}

func NormalizePorts(ports []int) []int {
	seen := make(map[int]struct{}, len(ports))
	out := make([]int, 0, len(ports))
	for _, port := range ports {
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		out = append(out, port)
	}
	sort.Ints(out)
	return out
}

func EncodePortBitmask(ports []int) int {
	mask := 0
	for _, port := range NormalizePorts(ports) {
		if port <= 0 {
			continue
		}
		mask |= 1 << (port - 1)
	}
	return mask
}
