package githubplan_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/githubplan"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
)

type nativeTextRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *nativeTextRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, request)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type nativeTextBody struct {
	data   []byte
	closed *bool
}

func (b *nativeTextBody) Read(destination []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(destination, b.data)
	b.data = b.data[n:]
	return n, nil
}
func (b *nativeTextBody) Close() error { *b.closed = true; return nil }

func TestNativeTitlesPreserveWhitespaceAndUnicodeExactly(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		root           string
		collection     string
	}{
		{
			name:           "milestone",
			representation: githubplan.MilestoneRepresentation,
			root:           `{"number":42,"title":"  計画 😀  ","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`,
			collection:     `[{"id":7,"title":"  作業 ©  "}]`,
		},
		{
			name:           "issue",
			representation: githubplan.IssueRepresentation,
			root:           `{"number":42,"title":"  計画 😀  ","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`,
			collection:     `[{"id":7,"title":"  作業 ©  "}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			transport := &nativeTextRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &nativeTextBody{data: []byte(tt.root), closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &nativeTextBody{data: []byte(tt.collection), closed: &collectionClosed}, Header: make(http.Header)},
			}}
			client := &http.Client{Transport: transport}
			repository := must(githubplan.NewRepository("owner", "repo"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			expected := must(plan.New(
				"  計画 😀  ",
				must(plan.NewGoal("Goal")),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))},
				[]plan.Task{must(plan.NewTask("  作業 ©  "))},
				nil,
			))
			controller := must(planrepresentation.NewController(observer))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.Satisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("differences mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.UnavailableInformation())); diff != "" {
				t.Errorf("unavailable information mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeBlankAndInvalidUTF8TitlesAreKnownPlanViolations(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		location       planrepresentation.LocationKind
		invalidUTF8    bool
		rootTitle      bool
	}{
		{name: "milestone blank root", representation: githubplan.MilestoneRepresentation, location: planrepresentation.PlanNameLocation, rootTitle: true},
		{name: "issue invalid UTF-8 root", representation: githubplan.IssueRepresentation, location: planrepresentation.PlanNameLocation, invalidUTF8: true, rootTitle: true},
		{name: "milestone invalid UTF-8 task", representation: githubplan.MilestoneRepresentation, location: planrepresentation.TaskCollectionLocation, invalidUTF8: true},
		{name: "issue blank task", representation: githubplan.IssueRepresentation, location: planrepresentation.TaskCollectionLocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentField := `"description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"`
			if tt.representation == githubplan.IssueRepresentation {
				contentField = `"body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"`
			}
			root := []byte(`{"number":42,"title":"Plan",` + contentField + `}`)
			collection := []byte(`[]`)
			if tt.rootTitle {
				root = []byte(`{"number":42,"title":" ",` + contentField + `}`)
				if tt.invalidUTF8 {
					root = append([]byte(`{"number":42,"title":"`), 0xff)
					root = append(root, []byte(`",`+contentField+`}`)...)
				}
			} else {
				collection = []byte(`[{"id":7,"title":" "}]`)
				if tt.invalidUTF8 {
					collection = append([]byte(`[{"id":7,"title":"`), 0xff)
					collection = append(collection, []byte(`"}]`)...)
				}
			}
			rootClosed, collectionClosed := false, false
			transport := &nativeTextRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &nativeTextBody{data: root, closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &nativeTextBody{data: collection, closed: &collectionClosed}, Header: make(http.Header)},
			}}
			client := &http.Client{Transport: transport}
			repository := must(githubplan.NewRepository("owner", "repo"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(observer))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, differences[0].Category()); diff != "" {
				t.Errorf("difference category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.location, differences[0].Location().Kind()); diff != "" {
				t.Errorf("difference location mismatch (-want +got):\n%s", diff)
			}
			invalid, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(plan.InvalidText, invalid.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.UnavailableInformation())); diff != "" {
				t.Errorf("unavailable information mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
