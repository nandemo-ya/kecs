package tui

import (
	"context"
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
		lbs, err := m.apiClient.ListLoadBalancers(ctx, m.selectedInstance)
		if err == nil {
			for _, lb := range lbs {
				loadBalancers = append(loadBalancers, LoadBalancer{
					ARN:       lb.LoadBalancerArn,
					Name:      lb.LoadBalancerName,
					DNSName:   lb.DNSName,
					Type:      lb.Type,
					Scheme:    lb.Scheme,
					State:     lb.State,
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
	switch msg.String() {
	case "tab":
		// Switch between sub-views
		m.elbv2SubView = (m.elbv2SubView + 1) % 3
		// Reset cursors when switching
		m.lbCursor = 0
		m.tgCursor = 0
		m.listenerCursor = 0

		// Load listeners when switching to listeners tab and a LB is selected
		if m.elbv2SubView == 2 && len(m.loadBalancers) > m.lbCursor {
			m.selectedLB = m.loadBalancers[m.lbCursor].ARN
			return m, m.loadListenersForLBCmd(m.selectedLB)
		}
		return m, nil

	case "up", "k":
		switch m.elbv2SubView {
		case 0: // Load Balancers
			if m.lbCursor > 0 {
				m.lbCursor--
			}
		case 1: // Target Groups
			if m.tgCursor > 0 {
				m.tgCursor--
			}
		case 2: // Listeners
			if m.listenerCursor > 0 {
				m.listenerCursor--
			}
		}
		return m, nil

	case "down", "j":
		switch m.elbv2SubView {
		case 0: // Load Balancers
			if m.lbCursor < len(m.loadBalancers)-1 {
				m.lbCursor++
			}
		case 1: // Target Groups
			if m.tgCursor < len(m.targetGroups)-1 {
				m.tgCursor++
			}
		case 2: // Listeners
			if m.listenerCursor < len(m.listeners)-1 {
				m.listenerCursor++
			}
		}
		return m, nil

	case "enter":
		// For load balancers, load listeners for selected LB
		if m.elbv2SubView == 0 && len(m.loadBalancers) > m.lbCursor {
			m.selectedLB = m.loadBalancers[m.lbCursor].ARN
			m.elbv2SubView = 2 // Switch to listeners view
			m.listenerCursor = 0
			return m, m.loadListenersForLBCmd(m.selectedLB)
		}
		return m, nil

	case "esc", "q":
		// Go back to clusters view
		m.currentView = ViewClusters
		return m, m.loadDataFromAPI()

	case "r":
		// Refresh data
		return m, m.loadELBv2DataCmd()

	case "/":
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case ":":
		m.commandMode = true
		m.commandInput = ""
		return m, nil
	}

	return m, nil
}
