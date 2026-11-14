package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SuggestionEvidence struct {
	Feature     string
	Choice      string
	Count       int
	Total       int
	Confidence  float64
	Specificity int
}

type LearnedSuggestion struct {
	Destination           string
	DestinationConfidence float64
	DestinationSupport    int
	DestinationReasons    []SuggestionEvidence
	RenameStyle           string
	RenameConfidence      float64
	RenameSupport         int
	RenameReasons         []SuggestionEvidence
}

func suggestionThreshold(feature string) (minCount int, minConfidence float64, specificity int) {
	switch {
	case strings.HasPrefix(feature, "pair:"):
		return 3, 0.67, 60
	case strings.HasPrefix(feature, "token:"):
		return 3, 0.70, 50
	case strings.HasPrefix(feature, "source:"):
		return 5, 0.75, 40
	case strings.HasPrefix(feature, "ext:"):
		return 5, 0.75, 30
	case strings.HasPrefix(feature, "type:"):
		return 8, 0.80, 20
	case feature == "all":
		return 12, 0.85, 10
	default:
		return 6, 0.80, 0
	}
}

func strongestChoiceForFeature(feature string, choices map[string]int) (SuggestionEvidence, bool) {
	total := learningFeatureTotal(choices)
	if total <= 0 {
		return SuggestionEvidence{}, false
	}
	winner := ""
	count := 0
	winners := 0
	for choice, n := range choices {
		if n > count {
			winner, count, winners = choice, n, 1
		} else if n == count && n > 0 {
			winners++
			if winner == "" || choice < winner {
				winner = choice
			}
		}
	}
	if winner == "" || count <= 0 || winners != 1 {
		return SuggestionEvidence{}, false
	}
	minCount, minConfidence, specificity := suggestionThreshold(feature)
	confidence := float64(count) / float64(total)
	if count < minCount || confidence < minConfidence {
		return SuggestionEvidence{}, false
	}
	return SuggestionEvidence{Feature: feature, Choice: winner, Count: count, Total: total, Confidence: confidence, Specificity: specificity}, true
}

func qualifyingSuggestionEvidence(features []string, table map[string]map[string]int, allow func(string) bool) []SuggestionEvidence {
	out := make([]SuggestionEvidence, 0)
	for _, feature := range features {
		choices := table[feature]
		if len(choices) == 0 {
			continue
		}
		evidence, ok := strongestChoiceForFeature(feature, choices)
		if !ok || (allow != nil && !allow(evidence.Choice)) {
			continue
		}
		out = append(out, evidence)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Specificity != out[j].Specificity {
			return out[i].Specificity > out[j].Specificity
		}
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Feature != out[j].Feature {
			return out[i].Feature < out[j].Feature
		}
		return out[i].Choice < out[j].Choice
	})
	return out
}

func chooseSuggestion(evidence []SuggestionEvidence) (choice string, confidence float64, support int, reasons []SuggestionEvidence) {
	if len(evidence) == 0 {
		return "", 0, 0, nil
	}
	choice = evidence[0].Choice
	for _, item := range evidence {
		if item.Choice != choice {
			continue
		}
		reasons = append(reasons, item)
		if item.Confidence > confidence {
			confidence = item.Confidence
		}
		if item.Count > support {
			support = item.Count
		}
		if len(reasons) >= 3 {
			break
		}
	}
	return choice, confidence, support, reasons
}

func buildLearnedSuggestion(model LearningModel, source string) LearnedSuggestion {
	normalizeLearningModel(&model)
	features := learningFeaturesForPath(source)

	destEvidence := qualifyingSuggestionEvidence(features, model.FeatureDestinations, func(choice string) bool {
		return strings.TrimSpace(choice) != ""
	})
	destination, destConfidence, destSupport, destReasons := chooseSuggestion(destEvidence)

	renameEvidence := qualifyingSuggestionEvidence(features, model.FeatureRenameStyles, func(choice string) bool {
		switch choice {
		case RenameStyleCleanSpacing, RenameStyleTitleCase, RenameStyleLower, RenameStyleUpper, RenameStyleDatePrefix:
			return true
		default:
			return false
		}
	})
	renameStyle, renameConfidence, renameSupport, renameReasons := chooseSuggestion(renameEvidence)

	return LearnedSuggestion{
		Destination:           destination,
		DestinationConfidence: destConfidence,
		DestinationSupport:    destSupport,
		DestinationReasons:    destReasons,
		RenameStyle:           renameStyle,
		RenameConfidence:      renameConfidence,
		RenameSupport:         renameSupport,
		RenameReasons:         renameReasons,
	}
}

func renameToolForLearnedStyle(style string) (string, bool) {
	switch style {
	case RenameStyleCleanSpacing:
		return RenameToolClean, true
	case RenameStyleTitleCase:
		return RenameToolTitleCase, true
	case RenameStyleLower:
		return RenameToolLower, true
	case RenameStyleUpper:
		return RenameToolUpper, true
	case RenameStyleDatePrefix:
		return RenameToolDatePrefix, true
	default:
		return "", false
	}
}

func proposedNameForRenameStyle(current, original, style string, fileDate time.Time) (string, error) {
	if style == RenameStyleTitleCase {
		result := titleCaseFilename(cleanRenameSpacing(current))
		if _, err := sanitizeBaseName(result); err != nil {
			return current, err
		}
		return result, nil
	}
	tool, ok := renameToolForLearnedStyle(style)
	if !ok {
		return current, fmt.Errorf("rename style %q cannot be applied automatically", style)
	}
	return applyRenameToolWithContext(current, original, tool, RenameToolContext{FileDate: fileDate, Template: DefaultRenameTemplate})
}

func applyLearnedSuggestionProposal(currentName, currentDestination string, suggestion LearnedSuggestion, fileDate time.Time) (string, string, error) {
	name := currentName
	dest := currentDestination
	if suggestion.RenameStyle != "" {
		var err error
		name, err = proposedNameForRenameStyle(currentName, currentName, suggestion.RenameStyle, fileDate)
		if err != nil {
			return currentName, currentDestination, err
		}
	}
	if suggestion.Destination != "" {
		dest = filepath.Clean(suggestion.Destination)
	}
	return name, dest, nil
}

func suggestionAvailable(s LearnedSuggestion) bool {
	return strings.TrimSpace(s.Destination) != "" || strings.TrimSpace(s.RenameStyle) != ""
}

func loadLearnedSuggestion(path, source string) (LearnedSuggestion, error) {
	model, err := loadLearningModel(path)
	if err != nil {
		return LearnedSuggestion{}, err
	}
	return buildLearnedSuggestion(model, source), nil
}

func suggestionSummaryConfidence(s LearnedSuggestion) float64 {
	if s.DestinationConfidence > s.RenameConfidence {
		return s.DestinationConfidence
	}
	return s.RenameConfidence
}
