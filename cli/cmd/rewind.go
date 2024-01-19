package cmd

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"strconv"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

// rewindCmd represents the rewind command
var rewindCmd = &cobra.Command{
	Use:     "rewind [steps-or-sha]",
	Aliases: []string{"rw"},
	Short:   "Rewind the plan to an earlier state",
	Long: `Rewind the plan to an earlier state.
	
	You can pass a "steps" number or a commit sha. If a steps number is passed, the plan will be rewound that many steps. If a commit sha is passed, the plan will be rewound to that commit. If neither a steps number nor a commit sha is passed, the target scope will be rewound by 1 step.
	`,
	Args: cobra.MaximumNArgs(1),
	Run:  rewind,
}

var sha string

func init() {
	// Add rewind command
	RootCmd.AddCommand(rewindCmd)

	// Add sha flag
	rewindCmd.Flags().StringVar(&sha, "sha", "", "Specify a commit sha to rewind to")
}

func rewind(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	// Check if either steps or sha is provided and not both
	stepsOrSha := ""
	if len(args) > 1 {
		stepsOrSha = args[1]
	}

	logsRes, err := api.Client.ListLogs(lib.CurrentPlanId)

	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
		return
	}

	var targetSha string

	log.Println("shas:", logsRes.Shas)

	if steps, err := strconv.Atoi(stepsOrSha); err == nil && steps > 0 {
		log.Println("steps:", steps)
		// Rewind by the specified number of steps
		targetSha = logsRes.Shas[steps]
	} else if sha := stepsOrSha; sha != "" {
		// Rewind to the specified Sha
		targetSha = sha
	} else if stepsOrSha == "" {
		// Rewind by 1 step
		targetSha = logsRes.Shas[1]
	} else {
		fmt.Fprintln(os.Stderr, "Invalid steps or sha. Steps must be a positive integer, and sha must be a valid commit hash.")
		os.Exit(1)
	}

	log.Println("Rewinding to", targetSha)

	// Rewind to the target sha
	rwRes, err := api.Client.RewindPlan(lib.CurrentPlanId, shared.RewindPlanRequest{Sha: targetSha})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rewinding plan: %v\n", err)
		return
	}

	msg := "✅ Rewound to " + targetSha

	fmt.Println(msg)
	fmt.Println()
	fmt.Println(rwRes.LatestCommit)
}
