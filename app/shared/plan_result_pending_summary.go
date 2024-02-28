package shared

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

func (state *CurrentPlanState) PendingChangesSummary() string {
	var msgs []string

	msgs = append(msgs, "🏗️  Build pending changes")

	descByConvoMessageId := make(map[string]*ConvoMessageDescription)
	for _, desc := range state.ConvoMessageDescriptions {
		if desc.ConvoMessageId == "" {
			log.Println("Warning: ConvoMessageId is empty for description:", desc)
			continue
		}

		descByConvoMessageId[desc.ConvoMessageId] = desc
	}

	type changeset struct {
		descsSet map[string]bool
		descs    []*ConvoMessageDescription
		results  []*PlanFileResult
	}
	byDescs := map[string]*changeset{}

	for _, result := range state.PlanResult.Results {
		// log.Println("result:")
		// spew.Dump(result)

		convoIds := map[string]bool{}
		if descByConvoMessageId[result.ConvoMessageId] != nil {
			convoIds[result.ConvoMessageId] = true
		}
		var uniqueConvoIds []string
		for convoId := range convoIds {
			uniqueConvoIds = append(uniqueConvoIds, convoId)
		}

		composite := strings.Join(uniqueConvoIds, "|")
		if _, ok := byDescs[composite]; !ok {
			byDescs[composite] = &changeset{
				descsSet: make(map[string]bool),
			}
		}

		ch := byDescs[composite]
		ch.results = append(byDescs[composite].results, result)

		// log.Println("uniqueConvoIds:", uniqueConvoIds)

		for _, convoMessageId := range uniqueConvoIds {
			if desc, ok := descByConvoMessageId[convoMessageId]; ok {
				if !ch.descsSet[convoMessageId] && !desc.DidBuild {
					ch.descs = append(ch.descs, desc)
					ch.descsSet[convoMessageId] = true
				}
			} else {
				log.Println("Warning: no description for convo message id:", convoMessageId)
			}
		}
	}

	var sortedChangesets []*changeset
	for _, ch := range byDescs {
		sortedChangesets = append(sortedChangesets, ch)
	}

	sort.Slice(sortedChangesets, func(i, j int) bool {
		// put changesets with no descriptions last, otherwise sort by date
		if len(sortedChangesets[i].descs) == 0 {
			return false
		}
		if len(sortedChangesets[j].descs) == 0 {
			return true
		}
		return sortedChangesets[i].descs[0].CreatedAt.Before(sortedChangesets[j].descs[0].CreatedAt)
	})

	for _, ch := range sortedChangesets {
		var descMsgs []string

		if len(ch.descs) == 0 {
			// log.Println("Warning: no descriptions for changeset")
			// spew.Dump(ch)
			continue
			// descMsgs = append(descMsgs, "  ✏️  Changes")
		}

		for _, desc := range ch.descs {
			descMsgs = append(descMsgs, fmt.Sprintf("  ✏️  %s", desc.CommitMsg))
		}

		pendingNewFilesSet := make(map[string]bool)
		pendingReplacementPathsSet := make(map[string]bool)
		pendingReplacementsByPath := make(map[string][]*Replacement)

		for _, result := range ch.results {

			if result.IsPending() {
				if len(result.Replacements) == 0 && result.Content != "" {
					pendingNewFilesSet[result.Path] = true
				} else {
					pendingReplacementPathsSet[result.Path] = true
					pendingReplacementsByPath[result.Path] = append(pendingReplacementsByPath[result.Path], result.Replacements...)
				}
			}
		}

		if len(pendingNewFilesSet) == 0 && len(pendingReplacementPathsSet) == 0 {
			continue
		}

		msgs = append(msgs, descMsgs...)

		var pendingNewFiles []string
		var pendingReplacementPaths []string

		for path := range pendingNewFilesSet {
			pendingNewFiles = append(pendingNewFiles, path)
		}

		for path := range pendingReplacementPathsSet {
			pendingReplacementPaths = append(pendingReplacementPaths, path)
		}

		sort.Slice(pendingReplacementPaths, func(i, j int) bool {
			return pendingReplacementPaths[i] < pendingReplacementPaths[j]
		})

		sort.Slice(pendingNewFiles, func(i, j int) bool {
			return pendingNewFiles[i] < pendingNewFiles[j]
		})

		if len(pendingNewFiles) > 0 {
			for _, path := range pendingNewFiles {
				msgs = append(msgs, fmt.Sprintf("    • new file → %s", path))
			}
		}

		if len(pendingReplacementPaths) > 0 {
			for _, path := range pendingReplacementPaths {
				msgs = append(msgs, fmt.Sprintf("    • edit → %s", path))
			}

		}

	}
	return strings.Join(msgs, "\n")
}
