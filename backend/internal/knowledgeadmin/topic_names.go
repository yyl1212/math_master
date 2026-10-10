package knowledgeadmin

import (
	"encoding/json"
	"github.com/yyl1212/math_master/schemas"
	"sort"
	"strings"
	"sync"
)

const directoryLabelsSHA = "3228ffa78270958036a43150002d628e2e1f816e7503ed9491a8b84ca90654a9"

type directoryLabel struct {
	English        string `json:"english"`
	Chinese        string `json:"chinese"`
	Kind           string `json:"kind"`
	DisplayEnglish string `json:"displayEnglish,omitempty"`
}

var labelsOnce sync.Once
var directoryLabels map[string]directoryLabel
var labelsError error

func loadDirectoryLabels() (map[string]directoryLabel, error) {
	labelsOnce.Do(func() {
		b, e := schemas.Files.ReadFile("topic-labels.zh-CN.json")
		if e != nil || DigestBytes(b) != directoryLabelsSHA {
			labelsError = ErrNotConfigured
			return
		}
		var doc struct {
			Entries map[string]directoryLabel `json:"entries"`
		}
		if json.Unmarshal(b, &doc) != nil || len(doc.Entries) != 6603 {
			labelsError = ErrNotConfigured
			return
		}
		for code, row := range doc.Entries {
			if code != strings.ToUpper(code) || strings.TrimSpace(row.English) == "" || strings.TrimSpace(row.Chinese) == "" {
				labelsError = ErrNotConfigured
				return
			}
		}
		directoryLabels = doc.Entries
	})
	return directoryLabels, labelsError
}

// Only an exact code, canonical English name and kind match may supply a missing translation.
func DirectoryTitles(code, english, chinese, kind string) (string, string, error) {
	labels, e := loadDirectoryLabels()
	if e != nil {
		return "", "", e
	}
	row, ok := labels[strings.ToUpper(code)]
	displayEnglish := english
	if ok && row.English == english && row.Kind == kind {
		if chinese == "" || chinese == english {
			chinese = row.Chinese
		}
		if row.DisplayEnglish != "" {
			displayEnglish = row.DisplayEnglish
		}
	}
	if chinese == "" {
		chinese = english
	}
	return chinese, displayEnglish, nil
}

// Return direct translated hits; the database must validate each hit before traversing ancestors.
func DirectoryChineseSearch(q string) (map[string]directoryLabel, error) {
	labels, e := loadDirectoryLabels()
	if e != nil {
		return nil, e
	}
	out := map[string]directoryLabel{}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return out, nil
	}
	for code, row := range labels {
		if !strings.Contains(strings.ToLower(row.Chinese), q) {
			continue
		}
		out[code] = row
	}
	return out, nil
}
func DirectoryChineseMatches(q string) ([]string, error) {
	matches, e := DirectoryChineseSearch(q)
	if e != nil {
		return nil, e
	}
	out := make([]string, 0, len(matches))
	for code := range matches {
		out = append(out, code)
	}
	sort.Strings(out)
	return out, nil
}
