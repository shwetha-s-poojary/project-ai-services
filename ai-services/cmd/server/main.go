package main

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	log "github.com/project-ai-services/ai-services/internal/pkg/server/logger"
	"github.com/project-ai-services/ai-services/internal/pkg/server/router"
)

var (
	servicePort = "8000"
)

func initFlags() {
	flag.StringVar(&servicePort, "port", "8000", "port to run the service on")
	flag.Parse()
}

func main() {
	logger := log.GetLogger()
	logger.Info("Starting ai-services server...")
	initFlags()
	// Entry point of the server application
	var appRouter = router.CreateRouter()
	logger.Info("ai-services server is up and running", zap.String("port", servicePort))
	logger.Fatal("Error encountered while routing", zap.Error(appRouter.Run(":"+servicePort)))
}
