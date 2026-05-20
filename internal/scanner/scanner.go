package scanner

import (
	"os"
	"sort"
	"sync"

	"github.com/mattn/go-runewidth"
	"github.com/phpgao/roost/internal/types"
)

func init() {
	// Force East Asian width so ellipsis "…" is consistently treated as 2 cols
	// across all locales (macOS vs Linux CI).
	_ = os.Setenv("RUNEWIDTH_EASTASIAN", "1")
	runewidth.DefaultCondition.EastAsianWidth = true
}

// Scanner defines the interface each platform scanner must implement.
type Scanner interface {
	Platform() types.Platform
	DataDir() string
	ScanProjects() ([]types.Project, error)
	DeleteSession(s types.Session) error
	DeleteProject(p types.Project) error
	ResumeCmd(s types.Session) []string
}

// mergeProjects merges []Project from multiple scanners by FullPath.
func mergeProjects(all [][]types.Project) []types.Project {
	m := make(map[string]*types.Project)
	var order []string
	for _, projects := range all {
		for _, p := range projects {
			if existing, ok := m[p.FullPath]; ok {
				existing.Sessions = append(existing.Sessions, p.Sessions...)
			} else {
				m[p.FullPath] = new(types.Project)
				*m[p.FullPath] = p
				order = append(order, p.FullPath)
			}
		}
	}
	result := make([]types.Project, 0, len(m))
	for _, key := range order {
		result = append(result, *m[key])
	}
	// Sort by last active time descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastActive().After(result[j].LastActive())
	})
	return result
}

// ScanProjectsParallel scans all platforms in parallel and returns merged results.
func ScanProjectsParallel(scanners []Scanner) []types.Project {
	var all [][]types.Project
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sc := range scanners {
		wg.Add(1)
		go func(sc Scanner) {
			defer wg.Done()
			projects, err := sc.ScanProjects()
			if err != nil || len(projects) == 0 {
				return
			}
			mu.Lock()
			all = append(all, projects)
			mu.Unlock()
		}(sc)
	}
	wg.Wait()

	return mergeProjects(all)
}
