package githubplan

import (
	"context"
	"net/http"

	"github.com/kotokumu/arcloom/planrepresentation"
)

const (
	InvalidClient         ViolationCode = "invalid_client"
	InvalidResourceNumber ViolationCode = "invalid_resource_number"
)

type observerBinding struct {
	repository Repository
	scheme     scheme
	number     ResourceNumber
}

type restAccess struct{ client http.Client }

// NewMilestoneObserver declares the concrete read Port for one Milestone.
func NewMilestoneObserver(client *http.Client, repository Repository, number ResourceNumber) (planrepresentation.Observer, error) {
	return newObserver(client, repository, MilestoneRepresentation, number)
}

// NewIssueObserver declares the concrete read Port for one parent Issue.
func NewIssueObserver(client *http.Client, repository Repository, number ResourceNumber) (planrepresentation.Observer, error) {
	return newObserver(client, repository, IssueRepresentation, number)
}

func newObserver(client *http.Client, repository Repository, representation Representation, number ResourceNumber) (planrepresentation.Observer, error) {
	if client == nil || client.Jar != nil {
		return nil, &ValidationError{code: InvalidClient, field: ClientField}
	}
	if !validRepository(repository) {
		return nil, &ValidationError{code: InvalidRepository, field: RepositoryOwnerField}
	}
	if number.value <= 0 {
		return nil, &ValidationError{code: InvalidResourceNumber, field: ResourceNumberField}
	}
	selected, valid := schemeFor(representation)
	if !valid {
		return nil, &ValidationError{code: InvalidRepresentation, field: RepresentationField}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	binding := observerBinding{
		repository: repository,
		scheme:     selected,
		number:     number,
	}
	access := restAccess{client: clientCopy}
	return func(ctx context.Context) (planrepresentation.Observation, error) {
		facts, err := access.observe(ctx, binding)
		if err != nil {
			return planrepresentation.Observation{}, err
		}
		return binding.scheme.project(facts)
	}, nil
}
