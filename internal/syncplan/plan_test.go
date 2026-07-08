package syncplan

import "testing"

func TestSummaryAdd(t *testing.T) {
	summary := Summary{}
	summary.Add(RepoPlan{Action: ActionReady})
	summary.Add(RepoPlan{Action: ActionSkipped})
	summary.Add(RepoPlan{Action: ActionFailed})

	if summary.Total != 3 || summary.Ready != 1 || summary.Skipped != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want one of each action", summary)
	}
}
