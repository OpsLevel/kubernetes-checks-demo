package agent

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/opslevel/kubernetes-checks-demo/config"

	"github.com/spf13/cobra"
	"github.com/rs/zerolog/log"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "",
	Long: ``,
	Run: doRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func doRun(cmd *cobra.Command, args []string) {
	c, err := config.New()
	cobra.CheckErr(err)
	startMessage()
	channel := setupSignalHandler()
	createControllers(channel, c)
	<-channel // Block until signals
	stopMessage()
}

var onlyOneSignalHandler = make(chan struct{})

func setupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func startMessage() {
	log.Info().Msg("Agent is Starting...")
}

func stopMessage() {
	log.Info().Msg("Agent is Stopping...")
}
