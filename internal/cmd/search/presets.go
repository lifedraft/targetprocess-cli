package search

import (
	"fmt"
	"sort"
)

// Preset defines a reusable search filter with optional field projection and sorting.
type Preset struct {
	Name        string
	Description string
	Where       string
	Select      string
	OrderBy     string
}

// SearchPresets is the map of all available search presets.
var SearchPresets = map[string]Preset{
	// Status-based
	"open": {
		Name:        "open",
		Description: "Entities in initial (open) state",
		Where:       "entityState.isInitial==true",
	},
	"inProgress": {
		Name:        "inProgress",
		Description: "Entities in a planned (in-progress) state",
		Where:       "entityState.isPlanned==true",
	},
	"done": {
		Name:        "done",
		Description: "Entities in final (done) state",
		Where:       "entityState.isFinal==true",
	},

	// Assignment-based
	"unassigned": {
		Name:        "unassigned",
		Description: "Entities with no assignments",
		Where:       "assignments.count==0",
	},

	// Priority-based
	"highPriority": {
		Name:        "highPriority",
		Description: "High-priority entities (importance >= 90)",
		Where:       "priority.importance>=90",
	},

	// Time-based
	"createdToday": {
		Name:        "createdToday",
		Description: "Entities created today",
		Where:       "createDate>=Today",
	},
	"modifiedToday": {
		Name:        "modifiedToday",
		Description: "Entities modified today",
		Where:       "modifyDate>=Today",
	},
	"createdThisWeek": {
		Name:        "createdThisWeek",
		Description: "Entities created in the last 7 days",
		Where:       "createDate>=Today.AddDays(-7)",
	},
	"modifiedThisWeek": {
		Name:        "modifiedThisWeek",
		Description: "Entities modified in the last 7 days",
		Where:       "modifyDate>=Today.AddDays(-7)",
	},
	"createdLastWeek": {
		Name:        "createdLastWeek",
		Description: "Entities created between 14 and 7 days ago",
		Where:       "createDate>=Today.AddDays(-14) and createDate<Today.AddDays(-7)",
	},
	"modifiedLastWeek": {
		Name:        "modifiedLastWeek",
		Description: "Entities modified between 14 and 7 days ago",
		Where:       "modifyDate>=Today.AddDays(-14) and modifyDate<Today.AddDays(-7)",
	},

	// Combined
	"highPriorityUnassigned": {
		Name:        "highPriorityUnassigned",
		Description: "High-priority entities with no assignments",
		Where:       "priority.importance>=90 and assignments.count==0",
	},

	// v2-powered presets with select and orderBy
	"recentActivity": {
		Name:        "recentActivity",
		Description: "Recently modified entities (last 7 days), sorted by modification date",
		Where:       "modifyDate>=Today.AddDays(-7)",
		Select:      "id,name,entityType.name as type,entityState.name as state,modifyDate",
		OrderBy:     "modifyDate desc",
	},
	"sprintStatus": {
		Name:        "sprintStatus",
		Description: "Active sprint items that are not yet done",
		Where:       "teamIteration!=null and entityState.isFinal!=true",
		Select:      "id,name,entityType.name as type,entityState.name as state,effort",
		OrderBy:     "entityState.name",
	},
	"unestimated": {
		Name:        "unestimated",
		Description: "Unestimated entities that are not yet done",
		Where:       "(effort==null or effort==0) and entityState.isFinal!=true",
		Select:      "id,name,entityState.name as state",
	},
}

// SortedPresetNames is the sorted list of preset names.
var SortedPresetNames = func() []string {
	names := make([]string, 0, len(SearchPresets))
	for name := range SearchPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}()

// ApplyPreset resolves a preset name into a full Preset struct.
// If where is also provided, the preset where and the extra where are combined with " and ".
func ApplyPreset(presetName, where string) (Preset, error) {
	p, ok := SearchPresets[presetName]
	if !ok {
		return Preset{}, fmt.Errorf("unknown preset %q, valid presets: %v", presetName, SortedPresetNames)
	}
	if where != "" {
		p.Where = p.Where + " and " + where
	}
	return p, nil
}
