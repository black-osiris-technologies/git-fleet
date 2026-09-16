package releaseflow

import (
	"fmt"
	"strings"
	"testing"
)

type tagRecordingRunner struct {
	calls  []string
	errors map[string]error
}

func (r *tagRecordingRunner) Run(_ string, command string, args ...string) (string, error) {
	key := strings.TrimSpace(command + " " + strings.Join(args, " "))
	r.calls = append(r.calls, key)
	if err := r.errors[key]; err != nil {
		return "", err
	}
	return "", nil
}

func (r *tagRecordingRunner) saw(fragment string) bool {
	for _, call := range r.calls {
		if strings.Contains(call, fragment) {
			return true
		}
	}
	return false
}

func TestCreateTagPublishesWithCreateOnlyLease(t *testing.T) {
	clone, _ := createTagRepo(t, nil, []string{"release-2.4"})
	runner := &tagRecordingRunner{}

	result := CreateTag(clone, TagOptions{}, runner)
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !runner.saw("git push --force-with-lease=refs/tags/v2.4.0: origin refs/tags/v2.4.0:refs/tags/v2.4.0") {
		t.Fatalf("CreateTag() calls = %#v, want create-only tag push", runner.calls)
	}
}

func TestCreateTagCleansLocalTagAfterPushFailure(t *testing.T) {
	clone, _ := createTagRepo(t, nil, []string{"release-2.4"})
	pushKey := "git push --force-with-lease=refs/tags/v2.4.0: origin refs/tags/v2.4.0:refs/tags/v2.4.0"
	runner := &tagRecordingRunner{errors: map[string]error{
		pushKey: fmt.Errorf("tag appeared concurrently"),
	}}

	result := CreateTag(clone, TagOptions{}, runner)
	if result.Action != ActionFailed {
		t.Fatalf("CreateTag() = %#v, want FAILED", result)
	}
	if !runner.saw("git tag -d v2.4.0") {
		t.Fatalf("CreateTag() calls = %#v, want local tag cleanup after failed publication", runner.calls)
	}
}
