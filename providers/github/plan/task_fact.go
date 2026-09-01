package githubplan

import (
	"encoding/json"
	"net/url"
	"strings"
)

// taskItemFact is the typed boundary result for one first-page collection
// item. REST id and title validity remain separate facts so one unusable item
// cannot widen a valid root or hide other members.
type taskItemFact struct {
	id          int64
	idValid     bool
	title       string
	titleValid  bool
	pullRequest bool
	state       nativeState
}

// taskFactSet is a per-observation collection of coherent page facts.
// It owns positive REST-id admission, same-id duplicate/conflict handling,
// page admission, visited-page detection, and membership completeness.
// Plan-location mapping remains in the selected representation decoder.
type taskFactSet struct {
	members      []taskItemFact
	complete     bool
	indexes      map[int64]int
	conflicts    map[int64]bool
	visitedPages map[int64]bool
}

type taskPageState uint8

const (
	taskPageUnavailable taskPageState = iota
	taskPageAvailable
)

type taskPageOutcome struct {
	state taskPageState
	items []taskItemFact
	next  nextPageOutcome
}

func decodeTaskFactPage(body []byte) ([]taskItemFact, bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed[0] != '[' {
		return nil, false
	}
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil || items == nil && trimmed != "[]" {
		return nil, false
	}
	facts := make([]taskItemFact, 0, len(items))
	for _, raw := range items {
		object, ok := decodeJSONObject(raw)
		if !ok {
			facts = append(facts, taskItemFact{})
			continue
		}
		fact := taskItemFact{}
		fact.id, fact.idValid = rawPositiveInt(object["id"])
		fact.title, fact.titleValid = rawString(object["title"])
		fact.state = decodeNativeState(object["state"])
		if marker, exists := object["pull_request"]; exists && strings.TrimSpace(string(marker)) != "null" {
			fact.pullRequest = true
		}
		facts = append(facts, fact)
	}
	return facts, true
}

func newTaskFactSet(items []taskItemFact, complete bool) taskFactSet {
	set := taskFactSet{
		complete:     complete,
		indexes:      make(map[int64]int, len(items)),
		conflicts:    make(map[int64]bool),
		visitedPages: make(map[int64]bool),
	}
	set.admit(items)
	return set
}

func (s *taskFactSet) admitPageNumber(value string) (int64, bool) {
	page, valid := positivePage(value)
	if !valid {
		s.markIncomplete()
		return 0, false
	}
	return page, s.beginPage(page)
}

func (s *taskFactSet) admitPage(outcome taskPageOutcome) (*url.URL, bool) {
	if outcome.state != taskPageAvailable {
		s.markIncomplete()
		return nil, false
	}
	s.admit(outcome.items)
	switch outcome.next.state {
	case nextPageAbsent:
		return nil, false
	case nextPageValid:
		return outcome.next.target, true
	default:
		s.markIncomplete()
		return nil, false
	}
}

func (s *taskFactSet) markIncomplete() { s.complete = false }

func (s taskFactSet) isComplete() bool { return s.complete }

func (s *taskFactSet) beginPage(page int64) bool {
	if page <= 0 || s.visitedPages[page] {
		s.complete = false
		return false
	}
	s.visitedPages[page] = true
	return true
}

func (s *taskFactSet) admit(items []taskItemFact) {
	for _, item := range items {
		if !item.idValid || !item.titleValid {
			s.complete = false
			continue
		}
		if s.conflicts[item.id] {
			s.complete = false
			continue
		}
		if index, exists := s.indexes[item.id]; exists {
			s.complete = false
			if !sameProgressFacts(s.members[index], item) {
				s.members = append(s.members[:index], s.members[index+1:]...)
				delete(s.indexes, item.id)
				for id, memberIndex := range s.indexes {
					if memberIndex > index {
						s.indexes[id] = memberIndex - 1
					}
				}
				s.conflicts[item.id] = true
			}
			continue
		}
		s.indexes[item.id] = len(s.members)
		s.members = append(s.members, item)
	}
}

func sameProgressFacts(left, right taskItemFact) bool {
	return left.title == right.title && left.state == right.state && left.pullRequest == right.pullRequest
}
