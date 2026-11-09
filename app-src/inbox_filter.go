package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type InboxSort string

const (
	InboxSortName                InboxSort = "Name"
	InboxSortFileType            InboxSort = "File Type"
	InboxSortDateCreated         InboxSort = "Date Created"
	InboxSortDateCreatedTypeName InboxSort = "Date Created > Type > Name"
)

var inboxSortOrder = []InboxSort{
	InboxSortName,
	InboxSortFileType,
	InboxSortDateCreated,
	InboxSortDateCreatedTypeName,
}

type InboxAgeFilter string

const (
	InboxAgeAll         InboxAgeFilter = "All ages"
	InboxAgeLast24Hours InboxAgeFilter = "Last 24 hours"
	InboxAgeLast7Days   InboxAgeFilter = "Last 7 days"
	InboxAgeLast30Days  InboxAgeFilter = "Last 30 days"
	InboxAgeOlder30Days InboxAgeFilter = "Older than 30 days"
)

var inboxAgeOrder = []InboxAgeFilter{
	InboxAgeAll,
	InboxAgeLast24Hours,
	InboxAgeLast7Days,
	InboxAgeLast30Days,
	InboxAgeOlder30Days,
}

type InboxSizeFilter string

const (
	InboxSizeAll        InboxSizeFilter = "All sizes"
	InboxSizeUnder1MiB  InboxSizeFilter = "Under 1 MiB"
	InboxSize1To10MiB   InboxSizeFilter = "1-10 MiB"
	InboxSize10To100MiB InboxSizeFilter = "10-100 MiB"
	InboxSize100MiBPlus InboxSizeFilter = "100 MiB+"
)

var inboxSizeOrder = []InboxSizeFilter{
	InboxSizeAll,
	InboxSizeUnder1MiB,
	InboxSize1To10MiB,
	InboxSize10To100MiB,
	InboxSize100MiBPlus,
}

type InboxFilter struct {
	Category     FileCategory
	Extension    string
	NameContains string
	Age          InboxAgeFilter
	Size         InboxSizeFilter
	Sort         InboxSort
}

func normalizedInboxFilter(filter InboxFilter) InboxFilter {
	if filter.Category == "" {
		filter.Category = CategoryAll
	}
	if strings.TrimSpace(filter.Extension) == "" {
		filter.Extension = "All extensions"
	}
	filter.NameContains = strings.TrimSpace(filter.NameContains)
	if filter.Age == "" {
		filter.Age = InboxAgeAll
	}
	if filter.Size == "" {
		filter.Size = InboxSizeAll
	}
	if filter.Sort == "" {
		filter.Sort = InboxSortDateCreated
	}
	return filter
}

func inboxAgeMatches(mod time.Time, ageFilter InboxAgeFilter, now time.Time) bool {
	if ageFilter == "" || ageFilter == InboxAgeAll {
		return true
	}
	age := now.Sub(mod)
	if age < 0 {
		age = 0
	}
	switch ageFilter {
	case InboxAgeLast24Hours:
		return age <= 24*time.Hour
	case InboxAgeLast7Days:
		return age <= 7*24*time.Hour
	case InboxAgeLast30Days:
		return age <= 30*24*time.Hour
	case InboxAgeOlder30Days:
		return age > 30*24*time.Hour
	default:
		return true
	}
}

func inboxSizeMatches(size int64, sizeFilter InboxSizeFilter) bool {
	const mib = int64(1024 * 1024)
	switch sizeFilter {
	case "", InboxSizeAll:
		return true
	case InboxSizeUnder1MiB:
		return size < mib
	case InboxSize1To10MiB:
		return size >= mib && size < 10*mib
	case InboxSize10To100MiB:
		return size >= 10*mib && size < 100*mib
	case InboxSize100MiBPlus:
		return size >= 100*mib
	default:
		return true
	}
}

func matchesInboxFilter(path string, info os.FileInfo, filter InboxFilter, now time.Time) bool {
	filter = normalizedInboxFilter(filter)
	if !matchesFilter(path, filter.Category, filter.Extension) {
		return false
	}
	if filter.NameContains != "" && !strings.Contains(strings.ToLower(filepath.Base(path)), strings.ToLower(filter.NameContains)) {
		return false
	}
	if info == nil {
		return false
	}
	if !inboxAgeMatches(info.ModTime(), filter.Age, now) {
		return false
	}
	return inboxSizeMatches(info.Size(), filter.Size)
}

func listFilesInbox(dir string, filter InboxFilter, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	filter = normalizedInboxFilter(filter)
	type candidate struct {
		path    string
		created time.Time
		ext     string
		name    string
	}
	items := make([]candidate, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if !matchesInboxFilter(path, info, filter, now) {
			continue
		}
		items = append(items, candidate{
			path:    path,
			created: fileCreatedTime(path, info),
			ext:     strings.ToLower(filepath.Ext(path)),
			name:    strings.ToLower(filepath.Base(path)),
		})
	}
	createdDate := func(t time.Time) (int, time.Month, int) {
		if t.IsZero() {
			return 0, 0, 0
		}
		return t.In(time.Local).Date()
	}
	newerCreated := func(a, b candidate) bool {
		if a.created.Equal(b.created) {
			return a.name < b.name
		}
		return a.created.After(b.created)
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch filter.Sort {
		case InboxSortName:
			return a.name < b.name
		case InboxSortFileType:
			if a.ext != b.ext {
				return a.ext < b.ext
			}
			return a.name < b.name
		case InboxSortDateCreatedTypeName:
			ay, am, ad := createdDate(a.created)
			by, bm, bd := createdDate(b.created)
			if ay != by {
				return ay > by
			}
			if am != bm {
				return am > bm
			}
			if ad != bd {
				return ad > bd
			}
			if a.ext != b.ext {
				return a.ext < b.ext
			}
			return a.name < b.name
		case InboxSortDateCreated:
			fallthrough
		default:
			return newerCreated(a, b)
		}
	})
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].path
	}
	return out, nil
}
