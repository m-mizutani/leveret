# Step Reflection

You need to reflect on the execution of an investigation step and determine if plan adjustments are needed.

## Step Information

**Step ID**: {{.StepID}}
**Description**: {{.StepDescription}}
**Expected Outcome**: {{.StepExpected}}

## Current Plan Status

**Already Completed Steps**:
{{if .CompletedSteps}}
{{range $index, $step := .CompletedSteps}}
{{add $index 1}}. {{$step}}
{{end}}
{{else}}
(None yet)
{{end}}

**Pending Steps**:
{{if .PendingSteps}}
{{range $index, $step := .PendingSteps}}
{{add $index 1}}. {{$step}}
{{end}}
{{else}}
(None)
{{end}}

Do not add steps that duplicate already completed or pending steps. Always check the lists above before adding new steps.

## Available Tools

When adding new steps, you must only use tools from this list:

{{range .AvailableTools}}
{{.}}
{{end}}

Do not create steps that use tools not in this list.

## Guidelines for Adding Steps

When considering adding new steps through reflection:

1. **Understand Tool Capabilities**:
   - Read tool descriptions carefully - tools can only do what they say
   - Do not ask tools to make judgments or decisions - they only retrieve data
   - Do not expect tools to access systems not mentioned in their description
   - Do not combine information from multiple sources in a single tool call

2. **What Tools CAN Do**:
   - Retrieve data (logs, threat intelligence, past alerts)
   - Execute queries (SQL, search, lookup)
   - Return structured information

3. **What Tools CANNOT Do**:
   - Make decisions or judgments
   - "Investigate" or "analyze" (they return raw data only)
   - Access systems outside their stated scope
   - Perform actions not mentioned in tool descriptions

4. **Keep Steps Focused**:
   - Combine related actions in one step when possible
   - Don't create separate steps for each indicator - check multiple indicators in one step
   - Don't create separate steps for "get schema" then "run query" - do both in one step

5. **Avoid Common Mistakes**:
   - Don't ask log tools to "determine if attack was successful" - they only retrieve logs
   - Don't ask threat intel tools to "investigate timeline" - they only provide reputation
   - Don't ask alert search to "analyze root cause" - it only finds similar alerts

## Execution Result

**Success**: {{.Success}}
**Findings**: {{.Findings}}

**Tools Used**:
{{range .ToolCalls}}
- {{.Name}}: {{.Result}}
{{end}}

## Reflection Questions

Evaluate the following:

1. **Was the expected outcome achieved?**
   - Did we get the information we needed?
   - Are there gaps in the findings?

2. **What new insights were discovered?**
   - Unexpected findings
   - New leads to follow
   - Contradictions or anomalies

3. **Should the plan be updated?**
   - Do we need additional steps?
   - Should any upcoming steps be modified?
   - Should any steps be skipped?

## Handling Failed Steps

Failed steps should be accepted, not retried. The initial plan should have included all necessary approaches.

**What NOT to do**:
- Create a new step that duplicates the failed step
- Use `add_step` to retry the same action with different parameters
- Try to use tools that don't exist or weren't in the available tools list
- Add steps to "try different approaches" - those should have been in the initial plan

**When a step fails**:

1. Analyze the error:
   - Missing tool/feature? Accept the limitation and move on
   - Insufficient data/logs? Accept the data gap and move on
   - System/infrastructure constraint? Accept the constraint and move on
   - Got partial information? Mark as achieved if the information is useful

2. Add new steps only if findings revealed new leads:
   - Step discovered a suspicious user account → Add step to investigate that account (using available tools only)
   - Step found anomalous timestamp → Add step to check logs around that time (using available tools only)
   - Do NOT add steps simply because the original approach didn't work
   - Do NOT add steps that require tools not in the "Available Tools" list

3. Accept limitations quickly:
   - Tool doesn't exist → Give up immediately, don't search for alternatives
   - Data source unavailable → Give up immediately, don't try other sources
   - Access denied → Give up immediately, don't try workarounds
   - No results found → Accept and move on, document the negative finding

## Response Format

Return your reflection in JSON format:

```json
{
  "achieved": true/false,
  "insights": [
    "New insight or discovery 1",
    "New insight or discovery 2"
  ],
  "plan_updates": [
    {
      "type": "add_step",
      "step": {
        "id": "step_2a",
        "description": "Description of additional investigation",
        "tools": ["tool_name"],
        "expected": "Expected outcome"
      }
    }
  ]
}
```

## Update Types

### add_step - Add New Step

Use this to add a new investigation step. The step will be appended at the end of the plan.

- `step`: Include complete step information (id, description, tools, expected)
- **IMPORTANT**: Only use tools from the "Available Tools" list above
- If you need a tool that's not available, do not create the step

Example:
```json
{
  "type": "add_step",
  "step": {
    "id": "step_3a",
    "description": "過去の類似アラートを検索",
    "tools": ["search_alerts"],
    "expected": "類似のアラートパターンと対応履歴"
  }
}
```

### update_step - Update Existing Step

Use this to modify an existing step.

- `step`: Include complete updated step information (use same ID as existing step)
- **IMPORTANT**: Only use tools from the "Available Tools" list above

Example:
```json
{
  "type": "update_step",
  "step": {
    "id": "step_4",
    "description": "より広い期間（7日間）のログをBigQueryで検索",
    "tools": ["bigquery_query"],
    "expected": "関連する全てのログイベント"
  }
}
```

### cancel_step - Cancel Step

Use this to cancel a step that is no longer needed.

- `step_id`: ID of the step to cancel
- `reason`: Explanation of why this step is no longer necessary

Example:
```json
{
  "type": "cancel_step",
  "step_id": "step_5",
  "reason": "This investigation path is unnecessary because the IP address was already confirmed benign in step_3"
}
```

## Common Scenarios

### Step Failed / Error Occurred

Failures should not trigger new steps. All approaches should have been planned upfront.

If the step failed due to tool error or unavailable data:
- Set `achieved: false`
- Add insight explaining what failed and why you're giving up
- Use empty `plan_updates` array - do not add retry or alternative steps

Examples of CORRECT handling:

**Tool doesn't exist:**
```json
{
  "achieved": false,
  "insights": ["ツールが存在しないため、この調査は実行できませんでした。別のアプローチは初期計画で検討すべきでした。"],
  "plan_updates": []
}
```

**No data found:**
```json
{
  "achieved": false,
  "insights": ["該当するログデータが見つかりませんでした。これ以上の調査は不可能です。"],
  "plan_updates": []
}
```

**Tool error:**
```json
{
  "achieved": false,
  "insights": ["API エラーが発生しました。システム制約のため、この情報源は利用できません。"],
  "plan_updates": []
}
```

### Step Partially Successful
- If the step got some useful information despite errors:
  - Set `achieved: true` (partial success is still progress)
  - Add insights about what was learned
  - Only add new steps if the findings revealed truly NEW leads (not alternatives to failures)

**IMPORTANT**:
- All text fields must be in Japanese (output language)
- Be conservative with plan updates - only suggest when truly necessary
- Focus on evidence gaps and new investigation paths
- If no updates needed, use empty array for plan_updates
- For update_step, provide complete step information (not partial modifications)
- **Never duplicate or retry the same step that just failed**
