# Rule Priority Management API Example

This example demonstrates how to use the KECS priority management functionality to automatically manage and optimize ELBv2 rule priorities.

## Priority Management Features

KECS provides several features to help manage rule priorities:

1. **Automatic Priority Assignment** - Find next available priority in a specific range
2. **Priority Validation** - Check if a priority is valid and available
3. **Priority Analysis** - Analyze rule distribution and detect conflicts
4. **Priority Optimization** - Suggest better priority assignments based on rule specificity
5. **Batch Priority Updates** - Update multiple rule priorities atomically

## Example Implementation

### 1. Automatic Priority Assignment

When creating a new rule, let KECS automatically assign an appropriate priority:

```go
// Example: Priority assignment service
package main

import (
    "context"
    "fmt"
    "github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
    "github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func assignPriorityForNewRule(ctx context.Context, store storage.ELBv2Store, listenerArn string, conditions []RuleCondition) (int32, error) {
    manager := elbv2.NewPriorityManager(store)
    
    // Calculate specificity to determine appropriate range
    specificity := calculateRuleSpecificity(conditions)
    
    var priorityRange elbv2.PriorityRange
    switch {
    case isHealthCheckRule(conditions):
        // Health checks get critical priority
        priorityRange = elbv2.PriorityRangeCritical
    case specificity >= 15:
        // Very specific rules (exact paths, multiple conditions)
        priorityRange = elbv2.PriorityRangeSpecific
    case specificity >= 5:
        // General application routes
        priorityRange = elbv2.PriorityRangeGeneral
    default:
        // Catch-all routes
        priorityRange = elbv2.PriorityRangeCatchAll
    }
    
    // Find next available priority in the range
    priority, err := manager.GetNextAvailablePriority(ctx, listenerArn, priorityRange)
    if err != nil {
        return 0, fmt.Errorf("failed to find available priority: %w", err)
    }
    
    return priority, nil
}
```

### 2. Priority Analysis and Optimization

Analyze existing rules and optimize their priorities:

```go
func optimizeRulePriorities(ctx context.Context, store storage.ELBv2Store, listenerArn string) error {
    manager := elbv2.NewPriorityManager(store)
    
    // Analyze current priority distribution
    analysis, err := manager.AnalyzeRulePriorities(ctx, listenerArn)
    if err != nil {
        return fmt.Errorf("failed to analyze priorities: %w", err)
    }
    
    fmt.Printf("Rule Priority Analysis:\n")
    fmt.Printf("Total Rules: %d\n", analysis.TotalRules)
    fmt.Printf("Distribution:\n")
    for range, count := range analysis.PriorityRanges {
        fmt.Printf("  %s: %d rules\n", range, count)
    }
    
    // Check for issues
    if len(analysis.Gaps) > 0 {
        fmt.Printf("\nLarge gaps detected:\n")
        for _, gap := range analysis.Gaps {
            fmt.Printf("  Gap between %d and %d (size: %d)\n", gap.Start-1, gap.End+1, gap.Size)
        }
    }
    
    if len(analysis.Conflicts) > 0 {
        fmt.Printf("\nPotential conflicts:\n")
        for _, conflict := range analysis.Conflicts {
            fmt.Printf("  %s\n", conflict.Message)
        }
    }
    
    // Get optimization suggestions
    suggestions, err := manager.OptimizePriorities(ctx, listenerArn)
    if err != nil {
        return fmt.Errorf("failed to get optimization suggestions: %w", err)
    }
    
    if len(suggestions) > 0 {
        fmt.Printf("\nOptimization suggestions:\n")
        for _, suggestion := range suggestions {
            fmt.Printf("  Move rule %s from priority %d to %d\n", 
                suggestion.RuleArn, getCurrentPriority(suggestion.RuleArn), suggestion.Priority)
        }
        
        // Apply suggestions
        if shouldApplyOptimizations() {
            err = manager.SetRulePriorities(ctx, suggestions)
            if err != nil {
                return fmt.Errorf("failed to apply optimizations: %w", err)
            }
            fmt.Println("Optimizations applied successfully")
        }
    }
    
    return nil
}
```

### 3. Rule Priority Validation

Validate priorities before creating or updating rules:

```go
func validateRulePriority(ctx context.Context, store storage.ELBv2Store, listenerArn string, priority int32, ruleArn string) error {
    manager := elbv2.NewPriorityManager(store)
    
    // Validate the priority
    err := manager.ValidatePriority(ctx, listenerArn, priority, ruleArn)
    if err != nil {
        // Priority is invalid or in use
        return err
    }
    
    // Additional business logic validation
    if priority < 100 && !isAdminUser() {
        return fmt.Errorf("priorities below 100 are reserved for system rules")
    }
    
    return nil
}
```

### 4. Batch Priority Updates

Update multiple rule priorities atomically:

```go
func reorganizeRulePriorities(ctx context.Context, store storage.ELBv2Store, listenerArn string) error {
    manager := elbv2.NewPriorityManager(store)
    
    // Option 1: Reorder with consistent gaps
    updates, err := manager.ReorderRulesForClarity(ctx, listenerArn, 10)
    if err != nil {
        return fmt.Errorf("failed to generate reorder plan: %w", err)
    }
    
    // Option 2: Manual priority updates
    manualUpdates := []elbv2.RulePriorityUpdate{
        {RuleArn: "arn:aws:elasticloadbalancing:us-east-1:123456789012:rule/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/8a2d6bc24d8a067", Priority: 10},
        {RuleArn: "arn:aws:elasticloadbalancing:us-east-1:123456789012:rule/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/9b3e7cd35e9b178", Priority: 20},
        {RuleArn: "arn:aws:elasticloadbalancing:us-east-1:123456789012:rule/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/0c4f8de46fac289", Priority: 30},
    }
    
    // Apply updates
    err = manager.SetRulePriorities(ctx, updates)
    if err != nil {
        return fmt.Errorf("failed to update priorities: %w", err)
    }
    
    return nil
}
```

### 5. Integration with Rule Creation

Integrate priority management when creating rules:

```go
func createRuleWithAutoPriority(ctx context.Context, storage storage.Storage, input *CreateRuleInput) (*Rule, error) {
    store := storage.ELBv2Store()
    manager := elbv2.NewPriorityManager(store)
    
    // If no priority specified, find appropriate one
    if input.Priority == nil {
        priority, err := manager.FindPriorityForConditions(ctx, input.ListenerArn, input.Conditions)
        if err != nil {
            return nil, fmt.Errorf("failed to determine priority: %w", err)
        }
        input.Priority = &priority
    } else {
        // Validate specified priority
        err := manager.ValidatePriority(ctx, input.ListenerArn, *input.Priority, "")
        if err != nil {
            return nil, fmt.Errorf("invalid priority: %w", err)
        }
    }
    
    // Create the rule
    rule := &storage.ELBv2Rule{
        ARN:         generateRuleArn(input.ListenerArn),
        ListenerArn: input.ListenerArn,
        Priority:    *input.Priority,
        Conditions:  serializeConditions(input.Conditions),
        Actions:     serializeActions(input.Actions),
    }
    
    err := store.CreateRule(ctx, rule)
    if err != nil {
        return nil, err
    }
    
    return rule, nil
}
```

## CLI Usage Examples

### Analyze Rule Priorities

```bash
# Get priority analysis for a listener
aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    | jq '.Rules | sort_by(.Priority) | group_by(.Priority / 1000 | floor) | map({range: (.[0].Priority / 1000 | floor), count: length, priorities: map(.Priority)})'
```

### Find Available Priority

```bash
# Find next available priority in specific range (100-999)
aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    | jq '[.Rules[].Priority] | sort | map(select(. >= 100 and . < 1000)) | . as $used | [range(100; 1000)] | map(select(. as $p | $used | index($p) | not)) | first'
```

### Batch Update Priorities

```bash
# Update multiple rule priorities
aws elbv2 set-rule-priorities \
    --rule-priorities \
        RuleArn=$RULE1_ARN,Priority=100 \
        RuleArn=$RULE2_ARN,Priority=200 \
        RuleArn=$RULE3_ARN,Priority=300 \
    --endpoint-url http://localhost:8080
```

## Best Practices

1. **Use Priority Ranges Consistently**
   - 1-99: Critical system routes (health checks, admin)
   - 100-999: Specific application routes
   - 1000-9999: General application routes
   - 10000+: Catch-all and default routes

2. **Leave Gaps Between Rules**
   - Use increments of 10 or 100
   - Makes it easier to insert new rules later

3. **Automate Priority Assignment**
   - Let KECS determine appropriate priorities based on rule specificity
   - Reduces human error and conflicts

4. **Regular Maintenance**
   - Periodically analyze and optimize rule priorities
   - Clean up gaps and reorganize as needed

5. **Monitor Priority Usage**
   - Track which priority ranges are filling up
   - Plan for expansion before running out of space

## Troubleshooting

### Priority Conflicts

If you get a "priority already in use" error:

1. Use the analysis API to see current priority usage
2. Find the next available priority in your desired range
3. Consider reorganizing existing rules if the range is full

### Performance Considerations

- Rules are evaluated in priority order (lowest first)
- Keep frequently-matched rules at lower priorities
- Use specific conditions to reduce evaluation overhead

### Debugging

Enable verbose logging to see priority decisions:

```bash
export KLOG_LEVEL=4
# Run your commands
```