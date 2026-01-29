package util

import (
	"dockflow/internal/domain"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// 从 172.30.0.0/16 切 /24：172.30.1.0/24 起
func AllocateSubnet(existing []domain.Namespace) (subnet, gateway string, err error) {
	used := map[int]bool{}
	for _, n := range existing {
		// subnet like 172.30.X.0/24
		x, ok := parseThirdOctet(n.Subnet)
		if ok {
			used[x] = true
		}
	}
	// 1..254
	for i := 1; i <= 254; i++ {
		if !used[i] {
			subnet = fmt.Sprintf("172.30.%d.0/24", i)
			gateway = fmt.Sprintf("172.30.%d.1", i)
			return
		}
	}
	return "", "", fmt.Errorf("no available subnet in 172.30.0.0/16")
}

func parseThirdOctet(subnet string) (int, bool) {
	// 172.30.X.0/24
	parts := strings.Split(subnet, "/")
	if len(parts) < 1 {
		return 0, false
	}
	ip := parts[0]
	oct := strings.Split(ip, ".")
	if len(oct) != 4 {
		return 0, false
	}
	if oct[0] != "172" || oct[1] != "30" {
		return 0, false
	}
	x, err := strconv.Atoi(oct[2])
	if err != nil {
		return 0, false
	}
	return x, true
}

// 可选：把 list 排序用
func SortByName(nss []domain.Namespace) {
	sort.Slice(nss, func(i, j int) bool {
		return nss[i].Name < nss[j].Name
	})
}
