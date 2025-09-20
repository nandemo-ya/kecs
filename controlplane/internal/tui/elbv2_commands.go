package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ELBv2 data loaded messages
type elbv2DataLoadedMsg struct {
	loadBalancers []LoadBalancer
	targetGroups  []TargetGroup
	listeners     []Listener
}

// loadELBv2DataCmd loads ELBv2 resources from the API
func (m Model) loadELBv2DataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var loadBalancers []LoadBalancer
		var targetGroups []TargetGroup
		var listeners []Listener

		// Load load balancers
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("loadELBv2DataCmd", "Loading load balancers for instance: %s", m.selectedInstance)
		}
		lbs, err := m.apiClient.ListLoadBalancers(ctx, m.selectedInstance)
		if err != nil {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadELBv2DataCmd", "Error loading load balancers: %v", err)
			}
		} else {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadELBv2DataCmd", "Loaded %d load balancers", len(lbs))
			}
			for _, lb := range lbs {
				// Extract state code from State object
				state := ""
				if lb.State != nil {
					state = lb.State.Code
				}
				loadBalancers = append(loadBalancers, LoadBalancer{
					ARN:       lb.LoadBalancerArn,
					Name:      lb.LoadBalancerName,
					DNSName:   lb.DNSName,
					Type:      lb.Type,
					Scheme:    lb.Scheme,
					State:     state,
					VpcID:     lb.VpcId,
					Subnets:   lb.Subnets,
					CreatedAt: lb.CreatedTime,
				})
			}
		}

		// Load target groups
		tgs, err := m.apiClient.ListTargetGroups(ctx, m.selectedInstance)
		if err == nil {
			for _, tg := range tgs {
				targetGroups = append(targetGroups, TargetGroup{
					ARN:                    tg.TargetGroupArn,
					Name:                   tg.TargetGroupName,
					Port:                   int(tg.Port),
					Protocol:               tg.Protocol,
					TargetType:             tg.TargetType,
					VpcID:                  tg.VpcId,
					HealthCheckEnabled:     tg.HealthCheckEnabled,
					HealthCheckPath:        tg.HealthCheckPath,
					HealthyTargetCount:     tg.HealthyTargetCount,
					UnhealthyTargetCount:   tg.UnhealthyTargetCount,
					RegisteredTargetsCount: tg.RegisteredTargetsCount,
				})
			}
		}

		// Load listeners for the first load balancer (if any)
		if len(loadBalancers) > 0 {
			lsts, err := m.apiClient.ListListeners(ctx, m.selectedInstance, loadBalancers[0].ARN)
			if err == nil {
				for _, lst := range lsts {
					var actions []ListenerAction
					for _, act := range lst.DefaultActions {
						actions = append(actions, ListenerAction{
							Type:           act.Type,
							TargetGroupArn: act.TargetGroupArn,
						})
					}
					listeners = append(listeners, Listener{
						ARN:             lst.ListenerArn,
						LoadBalancerARN: lst.LoadBalancerArn,
						Port:            int(lst.Port),
						Protocol:        lst.Protocol,
						DefaultActions:  actions,
						RuleCount:       0, // TODO: fetch rule count
					})
				}
			}
		}

		return elbv2DataLoadedMsg{
			loadBalancers: loadBalancers,
			targetGroups:  targetGroups,
			listeners:     listeners,
		}
	}
}

// loadListenersForLBCmd loads listeners for a specific load balancer
func (m Model) loadListenersForLBCmd(loadBalancerARN string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var listeners []Listener
		lsts, err := m.apiClient.ListListeners(ctx, m.selectedInstance, loadBalancerARN)
		if err == nil {
			for _, lst := range lsts {
				var actions []ListenerAction
				for _, act := range lst.DefaultActions {
					actions = append(actions, ListenerAction{
						Type:           act.Type,
						TargetGroupArn: act.TargetGroupArn,
					})
				}
				listeners = append(listeners, Listener{
					ARN:             lst.ListenerArn,
					LoadBalancerARN: lst.LoadBalancerArn,
					Port:            int(lst.Port),
					Protocol:        lst.Protocol,
					DefaultActions:  actions,
					RuleCount:       0, // TODO: fetch rule count
				})
			}
		}

		return elbv2DataLoadedMsg{
			listeners: listeners,
		}
	}
}

// handleELBv2Keys handles key events for ELBv2 views
func (m Model) handleELBv2Keys(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	// Debug logging - log every key press
	if debugLogger := GetDebugLogger(); debugLogger != nil {
		debugLogger.LogWithCaller("handleELBv2Keys", "=== Key event received ===")
		debugLogger.LogWithCaller("handleELBv2Keys", "Key: '%s', Type: %s, Runes: %v", key, msg.Type.String(), msg.Runes)
		debugLogger.LogWithCaller("handleELBv2Keys", "Current view: %s, LB cursor: %d, TG cursor: %d, Listener cursor: %d",
			m.currentView.String(), m.lbCursor, m.tgCursor, m.listenerCursor)
		debugLogger.LogWithCaller("handleELBv2Keys", "LoadBalancers count: %d, TargetGroups count: %d, Listeners count: %d",
			len(m.loadBalancers), len(m.targetGroups), len(m.listeners))
	}

	// Handle movement keys directly (they might not be in global actions for these views)
	switch key {
	case "up", "k":
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "Processing UP/K key - moving cursor up")
		}
		switch m.currentView {
		case ViewLoadBalancers:
			oldCursor := m.lbCursor
			if m.lbCursor > 0 {
				m.lbCursor--
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "LB cursor moved from %d to %d", oldCursor, m.lbCursor)
			}
		case ViewTargetGroups:
			oldCursor := m.tgCursor
			if m.tgCursor > 0 {
				m.tgCursor--
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TG cursor moved from %d to %d", oldCursor, m.tgCursor)
			}
		case ViewListeners:
			oldCursor := m.listenerCursor
			if m.listenerCursor > 0 {
				m.listenerCursor--
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Listener cursor moved from %d to %d", oldCursor, m.listenerCursor)
			}
		}
		return m, nil

	case "down", "j":
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "Processing DOWN/J key - moving cursor down")
		}
		switch m.currentView {
		case ViewLoadBalancers:
			oldCursor := m.lbCursor
			if m.lbCursor < len(m.loadBalancers)-1 {
				m.lbCursor++
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "LB cursor moved from %d to %d", oldCursor, m.lbCursor)
			}
		case ViewTargetGroups:
			oldCursor := m.tgCursor
			if m.tgCursor < len(m.targetGroups)-1 {
				m.tgCursor++
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TG cursor moved from %d to %d", oldCursor, m.tgCursor)
			}
		case ViewListeners:
			oldCursor := m.listenerCursor
			if m.listenerCursor < len(m.listeners)-1 {
				m.listenerCursor++
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Listener cursor moved from %d to %d", oldCursor, m.listenerCursor)
			}
		}
		return m, nil
	}

	// Check for global actions
	if action, found := m.keyBindings.GetGlobalAction(key); found {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "Global action found for key '%s': %s", key, action)
		}
		switch action {
		case ActionBack:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionBack - going to Clusters view")
			}
			// Go back to clusters view
			m.currentView = ViewClusters
			return m, m.loadDataFromAPI()
		case ActionGoHome:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionGoHome - going to Instances view")
			}
			// Go to home (instances view)
			m.currentView = ViewInstances
			m.selectedCluster = ""
			m.selectedService = ""
			return m, m.loadDataFromAPI()
		case ActionRefresh:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionRefresh - reloading ELBv2 data")
			}
			// Refresh data
			return m, m.loadELBv2DataCmd()
		case ActionSearch:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionSearch - entering search mode")
			}
			m.searchMode = true
			m.searchQuery = ""
			return m, nil
		case ActionCommand:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionCommand - entering command mode")
			}
			m.commandMode = true
			m.commandInput = ""
			return m, nil
		case ActionQuit:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionQuit - quitting application")
			}
			// Quit the application
			return m, tea.Quit
		}
	} else {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "No global action found for key '%s'", key)
		}
	}

	// Handle navigation keys explicitly
	switch key {
	case "c":
		// Navigate to clusters
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "C key pressed - Navigating to Clusters view")
		}
		m.currentView = ViewClusters
		return m, m.loadDataFromAPI()

	case "b":
		// Navigate to load balancers
		if m.currentView != ViewLoadBalancers {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "B key pressed - Navigating to LoadBalancers view")
			}
			m.currentView = ViewLoadBalancers
			m.lbCursor = 0
			return m, nil
		} else {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "B key pressed - Already in LoadBalancers view")
			}
		}

	case "g":
		// Navigate to target groups
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "G key pressed - Navigating to TargetGroups view")
		}
		m.currentView = ViewTargetGroups
		m.tgCursor = 0
		return m, nil

	case "enter":
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "ENTER key pressed in view: %s", m.currentView.String())
		}
		// Handle enter key based on current view
		switch m.currentView {
		case ViewLoadBalancers:
			// View listeners for selected load balancer
			if len(m.loadBalancers) > m.lbCursor {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "Navigating to Listeners for LB at index %d", m.lbCursor)
				}
				m.selectedLB = m.loadBalancers[m.lbCursor].ARN
				m.currentView = ViewListeners
				m.listenerCursor = 0
				return m, m.loadListenersForLBCmd(m.selectedLB)
			} else {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "No load balancer at cursor position %d", m.lbCursor)
				}
			}
		case ViewTargetGroups:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TODO: View targets for selected target group")
			}
			// TODO: View targets for selected target group
			return m, nil
		case ViewListeners:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TODO: View rules for selected listener")
			}
			// TODO: View rules for selected listener
			return m, nil
		}
		return m, nil

	case "y":
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "Y key pressed - Yanking ARN")
		}
		// Yank (copy) ARN
		var arn string
		switch m.currentView {
		case ViewLoadBalancers:
			if len(m.loadBalancers) > m.lbCursor {
				arn = m.loadBalancers[m.lbCursor].ARN
			}
		case ViewTargetGroups:
			if len(m.targetGroups) > m.tgCursor {
				arn = m.targetGroups[m.tgCursor].ARN
			}
		case ViewListeners:
			if len(m.listeners) > m.listenerCursor {
				arn = m.listeners[m.listenerCursor].ARN
			}
		}
		if arn != "" {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "ARN yanked: %s", arn)
			}
			// TODO: Implement clipboard functionality
			// For now, just acknowledge the action
			m.clipboardMsg = fmt.Sprintf("ARN copied: %s", arn)
			m.clipboardMsgTime = time.Now()
		} else {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "No ARN to yank at current position")
			}
		}
		return m, nil

	case "r":
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "R key pressed - Refreshing ELBv2 data")
		}
		// Refresh ELBv2 data
		return m, m.loadELBv2DataCmd()
	}

	// Check for view-specific actions
	if action, found := m.keyBindings.GetViewAction(m.currentView, key); found {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "View-specific action found for key '%s': %s", key, action)
		}
		switch action {
		case ActionNavigateListeners:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionNavigateListeners")
			}
			// For load balancers, load listeners for selected LB
			if m.currentView == ViewLoadBalancers && len(m.loadBalancers) > m.lbCursor {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "Loading listeners for LB at index %d", m.lbCursor)
				}
				m.selectedLB = m.loadBalancers[m.lbCursor].ARN
				m.currentView = ViewListeners
				m.listenerCursor = 0
				return m, m.loadListenersForLBCmd(m.selectedLB)
			}
			return m, nil

		case ActionNavigateTargetGroups:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionNavigateTargetGroups")
			}
			// Switch to target groups view
			m.currentView = ViewTargetGroups
			m.tgCursor = 0
			return m, nil

		case ActionNavigateLoadBalancers:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionNavigateLoadBalancers")
			}
			// Switch to load balancers view
			m.currentView = ViewLoadBalancers
			m.lbCursor = 0
			return m, nil

		case ActionNavigateClusters:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionNavigateClusters")
			}
			// Go back to clusters view
			m.currentView = ViewClusters
			return m, m.loadDataFromAPI()

		case ActionSelect:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionSelect in view: %s", m.currentView.String())
			}
			// View details or navigate deeper
			switch m.currentView {
			case ViewTargetGroups:
				// TODO: Show target group targets
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "TODO: Show target group targets")
				}
				return m, nil
			case ViewListeners:
				// TODO: Show listener rules
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "TODO: Show listener rules")
				}
				return m, nil
			}
			return m, nil

		case ActionYank:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Executing ActionYank (TODO)")
			}
			// Copy ARN to clipboard
			// TODO: Implement clipboard functionality
			return m, nil
		default:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "Unknown view-specific action: %s", action)
			}
		}
	} else {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "No view-specific action found for key '%s' in view %s", key, m.currentView.String())
		}
	}

	// Handle tab key for compatibility (if not in keybindings)
	if key == "tab" {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("handleELBv2Keys", "TAB key pressed - cycling through views")
		}
		// Cycle through ELBv2 views
		switch m.currentView {
		case ViewLoadBalancers:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TAB: LoadBalancers -> TargetGroups")
			}
			m.currentView = ViewTargetGroups
			m.tgCursor = 0
		case ViewTargetGroups:
			if len(m.loadBalancers) > 0 {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("handleELBv2Keys", "TAB: TargetGroups -> Listeners")
				}
				m.selectedLB = m.loadBalancers[0].ARN
				m.currentView = ViewListeners
				m.listenerCursor = 0
				return m, m.loadListenersForLBCmd(m.selectedLB)
			}
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TAB: TargetGroups -> LoadBalancers (no LBs for listeners)")
			}
			m.currentView = ViewLoadBalancers
			m.lbCursor = 0
		case ViewListeners:
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("handleELBv2Keys", "TAB: Listeners -> LoadBalancers")
			}
			m.currentView = ViewLoadBalancers
			m.lbCursor = 0
		}
		return m, nil
	}

	// Log unhandled key
	if debugLogger := GetDebugLogger(); debugLogger != nil {
		debugLogger.LogWithCaller("handleELBv2Keys", "Key '%s' not handled - no action taken", key)
	}

	return m, nil
}
