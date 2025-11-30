package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/tool"
)

// displayPlan shows the plan
func displayPlan(plan *Plan) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📌 計画\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("\n目的: %s\n\n", plan.Objective)
	fmt.Printf("ステップ:\n")
	for i, step := range plan.Steps {
		fmt.Printf("  %d. %s\n", i+1, step.Description)
		fmt.Printf("     期待: %s\n", step.Expected)
	}
	fmt.Printf("\n")
}

// executeStepsWithReflection executes all pending steps with reflection loop
func executeStepsWithReflection(
	ctx context.Context,
	gemini adapter.Gemini,
	registry *tool.Registry,
	plan *Plan,
) ([]*StepResult, []*Reflection, error) {
	results := make([]*StepResult, 0)
	reflections := make([]*Reflection, 0)

	// Execute steps dynamically - find next pending step each iteration
	for {
		// Find next pending step
		currentStepIndex := findNextPendingStep(plan)
		if currentStepIndex < 0 {
			break
		}

		// Execute the step
		result, err := executeStep(ctx, gemini, registry, plan, currentStepIndex, results)
		if err != nil {
			// Continue with failed result
			result = &StepResult{
				StepID:     plan.Steps[currentStepIndex].ID,
				Success:    false,
				Findings:   fmt.Sprintf("Step execution error: %v", err),
				ExecutedAt: time.Now(),
			}
		}
		results = append(results, result)

		// Mark as completed
		plan.Steps[currentStepIndex].Status = StepStatusCompleted

		// Reflect and potentially update plan
		reflection, err := reflectAndUpdatePlan(ctx, gemini, registry, plan, currentStepIndex, result)
		if err != nil {
			return nil, nil, err
		}
		reflections = append(reflections, reflection)
	}

	return results, reflections, nil
}

// findNextPendingStep finds the index of the next pending step
func findNextPendingStep(plan *Plan) int {
	for i, step := range plan.Steps {
		if step.Status == StepStatusPending {
			return i
		}
	}
	return -1
}

// executeStep executes a single step
func executeStep(
	ctx context.Context,
	gemini adapter.Gemini,
	registry *tool.Registry,
	plan *Plan,
	stepIndex int,
	previousResults []*StepResult,
) (*StepResult, error) {
	// Update status to in progress
	plan.Steps[stepIndex].Status = StepStatusInProgress

	// Display progress
	completedCount := countCompletedSteps(plan)
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🔍 ステップ %d/%d (完了: %d): %s\n", stepIndex+1, len(plan.Steps), completedCount, plan.Steps[stepIndex].ID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("%s\n\n", plan.Steps[stepIndex].Description)

	// Execute step
	result, err := runStepExecution(ctx, gemini, registry, &plan.Steps[stepIndex], plan, previousResults)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// countCompletedSteps counts how many steps are completed
func countCompletedSteps(plan *Plan) int {
	count := 0
	for _, step := range plan.Steps {
		if step.Status == StepStatusCompleted {
			count++
		}
	}
	return count
}

// reflectAndUpdatePlan reflects on step result and updates plan if needed
func reflectAndUpdatePlan(
	ctx context.Context,
	gemini adapter.Gemini,
	registry *tool.Registry,
	plan *Plan,
	stepIndex int,
	result *StepResult,
) (*Reflection, error) {
	// Reflect on result
	fmt.Printf("\n🤔 振り返り中...\n")
	reflection, err := reflect(ctx, gemini, registry, &plan.Steps[stepIndex], result, plan)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to reflect on step result")
	}

	// Display reflection insights
	displayReflectionInsights(reflection)

	// Apply plan updates if any
	if len(reflection.PlanUpdates) > 0 {
		if err := applyPlanUpdatesWithDisplay(plan, reflection); err != nil {
			return nil, err
		}
	}

	return reflection, nil
}

// displayReflectionInsights displays insights from reflection
func displayReflectionInsights(reflection *Reflection) {
	if len(reflection.Insights) > 0 {
		fmt.Printf("\n💡 新たな洞察:\n")
		for _, insight := range reflection.Insights {
			fmt.Printf("  - %s\n", insight)
		}
	}
}

// applyPlanUpdatesWithDisplay applies plan updates and displays the result
func applyPlanUpdatesWithDisplay(plan *Plan, reflection *Reflection) error {
	fmt.Printf("\n📝 計画を更新中...\n")
	if err := applyUpdates(plan, reflection); err != nil {
		return goerr.Wrap(err, "failed to apply plan updates")
	}
	fmt.Printf("   %d件の更新を適用しました\n\n", len(reflection.PlanUpdates))

	// Display updated plan status
	displayPlanStatus(plan)

	return nil
}

// displayPlanStatus shows completed, canceled, and pending steps
func displayPlanStatus(plan *Plan) {
	fmt.Printf("   ✅ 完了済み:\n")
	for idx, step := range plan.Steps {
		if step.Status == StepStatusCompleted {
			fmt.Printf("      %d. %s\n", idx+1, step.Description)
		}
	}

	// Show canceled steps if any
	hasCanceled := false
	for _, step := range plan.Steps {
		if step.Status == StepStatusCanceled {
			if !hasCanceled {
				fmt.Printf("\n   ❌ キャンセル済み:\n")
				hasCanceled = true
			}
			fmt.Printf("      %d. %s\n", 0, step.Description)
		}
	}

	fmt.Printf("\n   📋 未実行:\n")
	for idx, step := range plan.Steps {
		if step.Status == StepStatusPending {
			fmt.Printf("      %d. %s\n", idx+1, step.Description)
		}
	}
}

// displayConclusion shows the final conclusion
func displayConclusion(conclusion *Conclusion) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📝 結論を生成中...\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("%s\n\n", conclusion.Content)
}
