package main

import (
	command "github.com/eduardodbr/kubediff/internal/command"
	log "github.com/sirupsen/logrus"
)

func main() {
	rootCmd := command.Newkubediff()
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(command.NewImages())
	rootCmd.AddCommand(command.NewEnvs())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
