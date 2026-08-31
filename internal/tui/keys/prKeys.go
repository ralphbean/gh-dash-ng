package keys

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	log "charm.land/log/v2"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
)

type PRKeyMap struct {
	PrevSidebarTab       key.Binding
	NextSidebarTab       key.Binding
	Approve              key.Binding
	Assign               key.Binding
	Unassign             key.Binding
	Label                key.Binding
	Comment              key.Binding
	Diff                 key.Binding
	Checkout             key.Binding
	Close                key.Binding
	SummaryViewMore      key.Binding
	Ready                key.Binding
	Reopen               key.Binding
	Merge                key.Binding
	Update               key.Binding
	WatchChecks          key.Binding
	ApproveWorkflows     key.Binding
	ToggleSmartFiltering key.Binding
	ViewIssues           key.Binding
	Snooze               key.Binding
	TriageThreads        key.Binding
	TriageNextThread     key.Binding
	TriagePrevThread     key.Binding
	TriageResolve        key.Binding
	Star                 key.Binding
}

var PRKeys = PRKeyMap{
	PrevSidebarTab: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "previous sidebar tab"),
	),
	NextSidebarTab: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next sidebar tab"),
	),
	Approve: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "approve"),
	),
	Assign: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "assign"),
	),
	Unassign: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "unassign"),
	),
	Label: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "label"),
	),
	Comment: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "comment"),
	),
	Diff: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "diff"),
	),
	Checkout: key.NewBinding(
		key.WithKeys("C", "space"),
		key.WithHelp("C/Space", "checkout"),
	),
	Close: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "close"),
	),
	SummaryViewMore: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "expand description"),
	),
	Reopen: key.NewBinding(
		key.WithKeys("X"),
		key.WithHelp("X", "reopen"),
	),
	Ready: key.NewBinding(
		key.WithKeys("W"),
		key.WithHelp("W", "toggle draft status"),
	),
	Merge: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "merge"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update pr from base branch"),
	),
	WatchChecks: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "watch checks"),
	),
	ApproveWorkflows: key.NewBinding(
		key.WithKeys("V"),
		key.WithHelp("V", "approve all workflows"),
	),
	ToggleSmartFiltering: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle smart filtering"),
	),
	ViewIssues: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "switch to issues"),
	),
	Snooze: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "snooze"),
	),
	TriageThreads: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "triage review threads"),
	),
	TriageNextThread: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next thread"),
	),
	TriagePrevThread: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "previous thread"),
	),
	TriageResolve: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "resolve thread"),
	),
	Star: key.NewBinding(
		key.WithKeys("*"),
		key.WithHelp("*", "toggle star"),
	),
}

func PRFullHelp() []key.Binding {
	return []key.Binding{
		PRKeys.PrevSidebarTab,
		PRKeys.NextSidebarTab,
		PRKeys.Approve,
		PRKeys.Assign,
		PRKeys.Unassign,
		PRKeys.Label,
		PRKeys.Comment,
		PRKeys.Diff,
		PRKeys.Checkout,
		PRKeys.Close,
		PRKeys.Ready,
		PRKeys.Reopen,
		PRKeys.Merge,
		PRKeys.Update,
		PRKeys.WatchChecks,
		PRKeys.ApproveWorkflows,
		PRKeys.ToggleSmartFiltering,
		PRKeys.ViewIssues,
		PRKeys.Snooze,
		PRKeys.TriageThreads,
		PRKeys.Star,
	}
}

// PRTriageFullHelp returns the keybindings relevant while the review-thread
// triage workflow is active, in place of PRFullHelp's normal PR action
// list (which is inert while triaging). It reuses PRKeys.Comment's current
// key for display but with triage-specific help text, since replying to a
// thread - not commenting on the PR - is what that key does while triaging.
func PRTriageFullHelp() []key.Binding {
	reply := PRKeys.Comment
	reply.SetHelp(reply.Help().Key, "reply to thread")

	return []key.Binding{
		PRKeys.TriageNextThread,
		PRKeys.TriagePrevThread,
		reply,
		PRKeys.TriageResolve,
	}
}

func rebindPRKeys(keys []config.Keybinding) error {
	CustomPRBindings = []key.Binding{}

	for _, prKey := range keys {
		if prKey.Builtin == "" {
			// Handle custom commands
			if prKey.Command != "" {
				name := prKey.Name
				if prKey.Name == "" {
					name = config.TruncateCommand(prKey.Command)
				}

				customBinding := key.NewBinding(
					key.WithKeys(prKey.Key),
					key.WithHelp(prKey.Key, name),
				)

				CustomPRBindings = append(CustomPRBindings, customBinding)
			}
			continue
		}

		log.Debug("Rebinding PR key", "builtin", prKey.Builtin, "key", prKey.Key)

		var key *key.Binding

		switch prKey.Builtin {
		case "prevSidebarTab":
			key = &PRKeys.PrevSidebarTab
		case "nextSidebarTab":
			key = &PRKeys.NextSidebarTab
		case "approve":
			key = &PRKeys.Approve
		case "assign":
			key = &PRKeys.Assign
		case "unassign":
			key = &PRKeys.Unassign
		case "label":
			key = &PRKeys.Label
		case "comment":
			key = &PRKeys.Comment
		case "diff":
			key = &PRKeys.Diff
		case "checkout":
			key = &PRKeys.Checkout
		case "close":
			key = &PRKeys.Close
		case "ready":
			key = &PRKeys.Ready
		case "reopen":
			key = &PRKeys.Reopen
		case "merge":
			key = &PRKeys.Merge
		case "update":
			key = &PRKeys.Update
		case "watchChecks":
			key = &PRKeys.WatchChecks
		case "approveWorkflows":
			key = &PRKeys.ApproveWorkflows
		case "viewIssues":
			key = &PRKeys.ViewIssues
		case "summaryViewMore":
			key = &PRKeys.SummaryViewMore
		case "snooze":
			key = &PRKeys.Snooze
		case "triageThreads":
			key = &PRKeys.TriageThreads
		case "triageNextThread":
			key = &PRKeys.TriageNextThread
		case "triagePrevThread":
			key = &PRKeys.TriagePrevThread
		case "triageResolve":
			key = &PRKeys.TriageResolve
		case "star":
			key = &PRKeys.Star
		default:
			return fmt.Errorf("unknown built-in pr key: '%s'", prKey.Builtin)
		}

		key.SetKeys(prKey.Key)

		helpDesc := key.Help().Desc
		if prKey.Name != "" {
			helpDesc = prKey.Name
		}
		key.SetHelp(prKey.Key, helpDesc)
	}

	return nil
}
