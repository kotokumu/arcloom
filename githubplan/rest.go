package githubplan

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

type responseDocument struct{ bytes []byte }

type restReadState uint8

const (
	restReadUnavailable restReadState = iota
	restReadAvailable
)

type restReadOutcome struct {
	state    restReadState
	document responseDocument
	links    []string
}

// githubFactSet is one call-local collection of provider-native facts. It
// retains its immutable Binding so identity and correspondence cannot drift
// between pages or projection.
type githubFactSet struct {
	binding observerBinding
	root    rootFactOutcome
	tasks   taskFactSet
}

func newGitHubFactSet(binding observerBinding) githubFactSet {
	return githubFactSet{
		binding: binding,
		tasks:   newTaskFactSet(nil, true),
	}
}

func (s *githubFactSet) admitRoot(outcome rootFactOutcome) { s.root = outcome }

func (s *githubFactSet) admitTaskPageNumber(value string) (int64, bool) {
	return s.tasks.admitPageNumber(value)
}

func (s *githubFactSet) admitTaskPage(outcome taskPageOutcome) (*url.URL, bool) {
	return s.tasks.admitPage(outcome)
}

func (a restAccess) observe(ctx context.Context, binding observerBinding) (githubFactSet, error) {
	facts := newGitHubFactSet(binding)
	if ctx == nil {
		return facts, nil
	}
	root, err := a.readRoot(ctx, binding)
	if err != nil {
		return githubFactSet{}, err
	}
	facts.admitRoot(root)
	if !root.available {
		return facts, nil
	}

	target := binding.scheme.taskURL(binding)
	for {
		if err := ctx.Err(); err != nil {
			return githubFactSet{}, err
		}
		_, admitted := facts.admitTaskPageNumber(target.Query().Get("page"))
		if !admitted {
			return facts, nil
		}
		outcome, err := a.readTaskPage(ctx, binding, target)
		if err != nil {
			return githubFactSet{}, err
		}
		next, continueReading := facts.admitTaskPage(outcome)
		if !continueReading {
			return facts, nil
		}
		target = next
	}
}

func (a restAccess) readRoot(ctx context.Context, binding observerBinding) (rootFactOutcome, error) {
	outcome, err := a.read(ctx, binding.scheme.rootURL(binding))
	if err != nil || outcome.state != restReadAvailable {
		return rootFactOutcome{}, err
	}
	return binding.scheme.decodeRoot(outcome.document, binding), nil
}

func (a restAccess) readTaskPage(ctx context.Context, binding observerBinding, target *url.URL) (taskPageOutcome, error) {
	outcome, err := a.read(ctx, target)
	if err != nil {
		return taskPageOutcome{}, err
	}
	if outcome.state != restReadAvailable {
		return taskPageOutcome{state: taskPageUnavailable}, nil
	}
	items, valid := decodeTaskFactPage(outcome.document.bytes)
	if !valid {
		return taskPageOutcome{state: taskPageUnavailable}, nil
	}
	return taskPageOutcome{
		state: taskPageAvailable,
		items: items,
		next:  binding.scheme.nextPage(binding, target, outcome.links),
	}, nil
}

func (a restAccess) read(ctx context.Context, target *url.URL) (restReadOutcome, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return restReadOutcome{}, nil
	}
	request.Header.Set("Accept", "application/vnd.github.raw+json")
	request.Header.Set("X-GitHub-Api-Version", RESTAPIVersion)
	request.Header.Set("User-Agent", "arcloom")
	response, err := a.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return restReadOutcome{}, ctxErr
		}
		return restReadOutcome{}, nil
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return restReadOutcome{}, ctxErr
	}
	if readErr != nil || closeErr != nil {
		return restReadOutcome{}, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return restReadOutcome{}, nil
	}
	return restReadOutcome{
		state:    restReadAvailable,
		document: responseDocument{bytes: body},
		links:    append([]string(nil), response.Header.Values("Link")...),
	}, nil
}
