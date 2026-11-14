package main

import (
	"path/filepath"
	"testing"
	"time"
)

func suggestionModel() LearningModel {
	m := newLearningModel()
	return m
}

func addFeatureChoice(m *LearningModel, feature, choice string, count int, kind LearningPatternKind) {
	var target map[string]map[string]int
	if kind == LearningPatternRename {
		target = m.FeatureRenameStyles
	} else {
		target = m.FeatureDestinations
	}
	if target[feature] == nil {
		target[feature] = map[string]int{}
	}
	target[feature][choice] = count
}

func TestLearnedSuggestionRequiresMinimumEvidence(t *testing.T) {
	m := suggestionModel()
	dest := filepath.Join("C:", "Office", "Accounts")
	addFeatureChoice(&m, "token:invoice", dest, 2, LearningPatternDestination)
	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "invoice_2026.pdf"))
	if got.Destination != "" {
		t.Fatalf("expected no destination suggestion with only two observations, got %q", got.Destination)
	}
}

func TestLearnedSuggestionUsesSpecificFilenameEvidence(t *testing.T) {
	m := suggestionModel()
	accounts := filepath.Join("C:", "Office", "Accounts")
	general := filepath.Join("C:", "Documents")
	addFeatureChoice(&m, "all", general, 20, LearningPatternDestination)
	m.FeatureDestinations["all"][accounts] = 3
	addFeatureChoice(&m, "pair:invoice+vendor", accounts, 4, LearningPatternDestination)
	m.FeatureDestinations["pair:invoice+vendor"][general] = 1
	addFeatureChoice(&m, "token:invoice", accounts, 8, LearningPatternDestination)
	m.FeatureDestinations["token:invoice"][general] = 2

	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "Vendor_Invoice_September.pdf"))
	if got.Destination != accounts {
		t.Fatalf("expected specific destination %q, got %q", accounts, got.Destination)
	}
	if got.DestinationConfidence < 0.79 {
		t.Fatalf("expected strong confidence, got %.3f", got.DestinationConfidence)
	}
	if len(got.DestinationReasons) == 0 || got.DestinationReasons[0].Feature != "pair:invoice+vendor" {
		t.Fatalf("expected pair evidence first, got %#v", got.DestinationReasons)
	}
}

func TestLearnedSuggestionSuppressesAmbiguousEvidence(t *testing.T) {
	m := suggestionModel()
	a := filepath.Join("C:", "A")
	b := filepath.Join("C:", "B")
	addFeatureChoice(&m, "token:report", a, 3, LearningPatternDestination)
	m.FeatureDestinations["token:report"][b] = 2
	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "report.pdf"))
	if got.Destination != "" {
		t.Fatalf("ambiguous 3/5 token evidence should stay silent, got %q", got.Destination)
	}
}

func TestLearnedSuggestionSuppressesTiedWinner(t *testing.T) {
	m := suggestionModel()
	a := filepath.Join("C:", "A")
	b := filepath.Join("C:", "B")
	addFeatureChoice(&m, "token:invoice", a, 4, LearningPatternDestination)
	m.FeatureDestinations["token:invoice"][b] = 4
	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "invoice.pdf"))
	if got.Destination != "" {
		t.Fatalf("tied top choices must stay silent, got %q", got.Destination)
	}
}

func TestLearnedSuggestionCanProposeKnownRenameStyle(t *testing.T) {
	m := suggestionModel()
	addFeatureChoice(&m, "token:sanction", RenameStyleDatePrefix, 4, LearningPatternRename)
	m.FeatureRenameStyles["token:sanction"][RenameStyleUnchanged] = 1
	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "sanction_order.pdf"))
	if got.RenameStyle != RenameStyleDatePrefix {
		t.Fatalf("expected date-prefix rename suggestion, got %q", got.RenameStyle)
	}
	name, err := proposedNameForRenameStyle("sanction order", "sanction order", got.RenameStyle, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if name != "2026-09-22 sanction order" {
		t.Fatalf("unexpected proposed name %q", name)
	}
}

func TestLearnedSuggestionDoesNotAutoApplyCustomRenameStyle(t *testing.T) {
	m := suggestionModel()
	addFeatureChoice(&m, "token:invoice", RenameStyleCustom, 10, LearningPatternRename)
	got := buildLearnedSuggestion(m, filepath.Join("C:", "Downloads", "invoice.pdf"))
	if got.RenameStyle != "" {
		t.Fatalf("custom rename transformations are not reproducible and must not be suggested, got %q", got.RenameStyle)
	}
}

func TestProposedSuggestionValuesDoNotTouchSourceOrModel(t *testing.T) {
	m := suggestionModel()
	dest := filepath.Join("C:", "Office", "5th SFC")
	addFeatureChoice(&m, "token:sfc", dest, 5, LearningPatternDestination)
	addFeatureChoice(&m, "token:sfc", RenameStyleTitleCase, 5, LearningPatternRename)
	source := filepath.Join("C:", "Downloads", "5th_sfc_order.pdf")
	before := m.FeatureDestinations["token:sfc"][dest]
	s := buildLearnedSuggestion(m, source)
	name, proposedDest, err := applyLearnedSuggestionProposal("5th_sfc_order", filepath.Join("C:", "Documents"), s, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if name != "5Th Sfc Order" {
		t.Fatalf("unexpected title-case proposal %q", name)
	}
	if proposedDest != dest {
		t.Fatalf("unexpected destination proposal %q", proposedDest)
	}
	if m.FeatureDestinations["token:sfc"][dest] != before {
		t.Fatal("building/applying a proposal must not train or mutate the learning model")
	}
}
