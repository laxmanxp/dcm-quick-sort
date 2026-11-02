package main

import (
	"sort"
)

type LearningPatternKind string

const (
	LearningPatternDestination LearningPatternKind = "destination"
	LearningPatternRename      LearningPatternKind = "rename"
)

type LearningPatternItem struct {
	Kind       LearningPatternKind
	Feature    string
	Choice     string
	Count      int
	Total      int
	Confidence float64
}

type LearningCenterState struct {
	TotalObservations int
	Paused            bool
}

func loadLearningCenterItems(path string) ([]LearningPatternItem, LearningCenterState, error) {
	model, err := loadLearningModel(path)
	if err != nil {
		return nil, LearningCenterState{}, err
	}
	items := make([]LearningPatternItem, 0)
	for feature, choices := range model.FeatureDestinations {
		total := learningFeatureTotal(choices)
		if total <= 0 {
			continue
		}
		for choice, count := range choices {
			if count <= 0 {
				continue
			}
			items = append(items, LearningPatternItem{Kind: LearningPatternDestination, Feature: feature, Choice: choice, Count: count, Total: total, Confidence: float64(count) / float64(total)})
		}
	}
	for feature, choices := range model.FeatureRenameStyles {
		total := learningFeatureTotal(choices)
		if total <= 0 {
			continue
		}
		for choice, count := range choices {
			if count <= 0 {
				continue
			}
			items = append(items, LearningPatternItem{Kind: LearningPatternRename, Feature: feature, Choice: choice, Count: count, Total: total, Confidence: float64(count) / float64(total)})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		if items[i].Confidence != items[j].Confidence {
			return items[i].Confidence > items[j].Confidence
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Feature != items[j].Feature {
			return items[i].Feature < items[j].Feature
		}
		return items[i].Choice < items[j].Choice
	})
	return items, LearningCenterState{TotalObservations: model.TotalObservations, Paused: model.Paused}, nil
}
