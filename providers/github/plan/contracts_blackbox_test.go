package githubplan_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type recordingRoundTripper struct{ requests *int }

func (r recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	*r.requests++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func TestNewResourceNumberBlackBox(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		wantValue int64
		wantErr   bool
		wantCode  githubplan.ViolationCode
		wantField githubplan.Field
	}{
		{name: "negative", value: -1, wantErr: true, wantCode: githubplan.InvalidResourceNumber, wantField: githubplan.ResourceNumberField},
		{name: "zero", value: 0, wantErr: true, wantCode: githubplan.InvalidResourceNumber, wantField: githubplan.ResourceNumberField},
		{name: "smallest positive", value: 1, wantValue: 1},
		{name: "largest positive", value: 9223372036854775807, wantValue: 9223372036854775807},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewResourceNumber(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewResourceNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var validation *githubplan.ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("validation = %v", err)
				}
				if diff := cmp.Diff(tt.wantCode, validation.Code()); diff != "" {
					t.Errorf("code mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tt.wantField, validation.Field()); diff != "" {
					t.Errorf("field mismatch (-want +got):\n%s", diff)
				}
				return
			}
			if diff := cmp.Diff(tt.wantValue, got.Int64()); diff != "" {
				t.Errorf("value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewMilestoneObserverContract(t *testing.T) {
	repository := must(githubplan.NewRepository("owner", "repo"))
	number := must(githubplan.NewResourceNumber(42))
	var zeroNumber githubplan.ResourceNumber
	tests := []struct {
		name         string
		client       *http.Client
		repository   githubplan.Repository
		number       githubplan.ResourceNumber
		wantObserver bool
		wantCode     githubplan.ViolationCode
		wantField    githubplan.Field
	}{
		{name: "valid binding", client: &http.Client{}, repository: repository, number: number, wantObserver: true},
		{name: "nil client", client: nil, repository: repository, number: number, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
		{name: "cookie jar", client: &http.Client{Jar: must(cookiejar.New(nil))}, repository: repository, number: number, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
		{name: "zero repository", client: &http.Client{}, repository: githubplan.Repository{}, number: number, wantCode: githubplan.InvalidRepository, wantField: githubplan.RepositoryOwnerField},
		{name: "zero resource number", client: &http.Client{}, repository: repository, number: zeroNumber, wantCode: githubplan.InvalidResourceNumber, wantField: githubplan.ResourceNumberField},
		{name: "all invalid precedence", client: nil, repository: githubplan.Repository{}, number: zeroNumber, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewMilestoneObserver(tt.client, tt.repository, tt.number)
			if (got != nil) != tt.wantObserver {
				t.Fatalf("observer presence = %v, want %v (err=%v)", got != nil, tt.wantObserver, err)
			}
			if tt.wantObserver {
				if err != nil {
					t.Fatalf("NewMilestoneObserver() error = %v", err)
				}
				return
			}
			var validation *githubplan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(tt.wantCode, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantField, validation.Field()); diff != "" {
				t.Errorf("field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewIssueObserverContract(t *testing.T) {
	repository := must(githubplan.NewRepository("owner", "repo"))
	number := must(githubplan.NewResourceNumber(42))
	var zeroNumber githubplan.ResourceNumber
	tests := []struct {
		name         string
		client       *http.Client
		repository   githubplan.Repository
		number       githubplan.ResourceNumber
		wantObserver bool
		wantCode     githubplan.ViolationCode
		wantField    githubplan.Field
	}{
		{name: "valid binding", client: &http.Client{}, repository: repository, number: number, wantObserver: true},
		{name: "nil client", client: nil, repository: repository, number: number, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
		{name: "cookie jar", client: &http.Client{Jar: must(cookiejar.New(nil))}, repository: repository, number: number, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
		{name: "zero repository", client: &http.Client{}, repository: githubplan.Repository{}, number: number, wantCode: githubplan.InvalidRepository, wantField: githubplan.RepositoryOwnerField},
		{name: "zero resource number", client: &http.Client{}, repository: repository, number: zeroNumber, wantCode: githubplan.InvalidResourceNumber, wantField: githubplan.ResourceNumberField},
		{name: "all invalid precedence", client: nil, repository: githubplan.Repository{}, number: zeroNumber, wantCode: githubplan.InvalidClient, wantField: githubplan.ClientField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewIssueObserver(tt.client, tt.repository, tt.number)
			if (got != nil) != tt.wantObserver {
				t.Fatalf("observer presence = %v, want %v (err=%v)", got != nil, tt.wantObserver, err)
			}
			if tt.wantObserver {
				if err != nil {
					t.Fatalf("NewIssueObserver() error = %v", err)
				}
				return
			}
			var validation *githubplan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(tt.wantCode, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantField, validation.Field()); diff != "" {
				t.Errorf("field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserverConstructionDoesNotMutateClient(t *testing.T) {
	repository := must(githubplan.NewRepository("owner", "repo"))
	number := must(githubplan.NewResourceNumber(42))
	sentinel := errors.New("sentinel")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return sentinel }}
	_, _ = githubplan.NewMilestoneObserver(client, repository, number)
	if client.CheckRedirect == nil {
		t.Fatal("NewMilestoneObserver() mutated CheckRedirect to nil")
	}
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, sentinel) {
		t.Fatalf("CheckRedirect() error = %v, want %v", err, sentinel)
	}
}

func TestObserverConstructionMakesNoRequest(t *testing.T) {
	repository := must(githubplan.NewRepository("owner", "repo"))
	number := must(githubplan.NewResourceNumber(42))
	requests := 0
	client := &http.Client{Transport: recordingRoundTripper{requests: &requests}}
	observer, err := githubplan.NewMilestoneObserver(client, repository, number)
	if observer == nil || err != nil {
		t.Errorf("NewMilestoneObserver() = (%v, %v), want a non-nil observer and nil error", observer != nil, err)
	}
	if diff := cmp.Diff(0, requests); diff != "" {
		t.Errorf("construction requests mismatch (-want +got):\n%s", diff)
	}
}
