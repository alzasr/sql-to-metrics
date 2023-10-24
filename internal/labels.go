package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"strings"
)

type Labels prometheus.Labels

func fromRes(res map[string]*string, labelNames []string) Labels {
	labels := make(Labels, len(labelNames))
	for _, name := range labelNames {
		labels[name] = ""
		if val := res[name]; val != nil {
			labels[name] = *val
		}
	}
	return labels
}

func (l Labels) String() string {
	keys := make([]string, 0, len(l))
	for key, _ := range l {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		keys[i] = keys[i] + ":" + l[key]
	}
	return strings.Join(keys, ";")
}
