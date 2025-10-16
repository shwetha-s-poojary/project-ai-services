package handlers

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/project-ai-services/ai-services/internal/pkg/helpers"
	"github.com/project-ai-services/ai-services/internal/pkg/logger"
	"github.com/project-ai-services/ai-services/internal/pkg/runtime"
	"github.com/project-ai-services/ai-services/internal/pkg/runtime/podman"
	"github.com/project-ai-services/ai-services/internal/pkg/server/models"
	"github.com/project-ai-services/ai-services/internal/pkg/utils"
)

const (
	servicesPath = "assets/services/"
)

var log *zap.Logger

func init() {
	log = logger.GetLogger()
}

type servicesHandler struct {
	// container Runtime
	runtime runtime.Runtime
}

func NewServicesHandler() *servicesHandler {
	runtime, err := podman.NewPodmanClient()
	if err != nil {
		panic(fmt.Sprintf("failed connecting to container runtime: %s", err.Error()))
	}
	return &servicesHandler{runtime: runtime}
}

// GetStats - returns the information about the deployed services
func (s *servicesHandler) GetStats(c *gin.Context) {
	resp, err := s.runtime.ListPods()
	if err != nil {
		log.Error("GET Services Stats failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResp{Error: models.Error{Code: http.StatusInternalServerError, Message: "Something went wrong. Please try again!"}})
	}

	var pods []*types.ListPodsReport
	if val, ok := resp.([]*types.ListPodsReport); ok {
		pods = val
	}

	convertToServiceObj := func(pods []*types.ListPodsReport) []models.Service {
		output := make([]models.Service, len(pods))
		for i, pod := range pods {
			output[i] = models.Service{ID: pod.Id, Name: pod.Name, Status: pod.Status}
		}
		return output
	}

	c.JSON(http.StatusOK, models.GetServicesResp{Services: convertToServiceObj(pods)})
}

// GetOffered - returns the list of offered services
func (s *servicesHandler) GetOffered(c *gin.Context) {
	// Read all the template files from the path
	templateFileNames, err := helpers.GetTemplateNames(servicesPath)
	if err != nil {
		log.Error("GET Services failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResp{Error: models.Error{Code: http.StatusInternalServerError, Message: "Something went wrong. Please try again!"}})
	}

	// Reformat the names
	var resp []string
	for _, templateFileName := range templateFileNames {
		resp = append(resp, utils.CapitalizeAndFormat(templateFileName))
	}

	c.JSON(http.StatusOK, resp)
}

func getReqServiceTemplatePath(serviceName string) (string, error) {
	var reqFileName string
	// Read all the template files from the path
	templateFileNames, err := helpers.GetTemplateNames(servicesPath)
	if err != nil {
		return reqFileName, err
	}

	if !slices.Contains(templateFileNames, serviceName) {
		return reqFileName, fmt.Errorf("the template for requested service: %s does not exist", serviceName)
	} else {
		// if present, append .yaml.tmpl
		reqFileName = fmt.Sprintf("%s.yaml.tmpl", serviceName)
	}

	return servicesPath + reqFileName, nil
}

// Deploy - Deploys a given service
func (s *servicesHandler) Deploy(c *gin.Context) {
	var serviceReq models.ServiceDeployReq

	if err := c.ShouldBindJSON(&serviceReq); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp{Error: models.Error{Code: http.StatusBadRequest, Message: err.Error()}})
		return
	}

	serviceTemplateFilePath, err := getReqServiceTemplatePath(serviceReq.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResp{Error: models.Error{Code: http.StatusBadRequest, Message: err.Error()}})
		return
	}

	params := map[string]any{
		"PodName":  serviceReq.Name,   // Replacing Podname with service name with some UUID to ensure uniqueness, for now going with the serviceName
		"HostPort": 8080,              // This might not be required, just added here for testing
		"Env":      serviceReq.Params, // Read from the request params
	}

	if err = s.runtime.CreatePodFromTemplate(serviceTemplateFilePath, params); err != nil {
		log.Error("Deploy Services failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResp{Error: models.Error{Code: http.StatusInternalServerError, Message: "Something went wrong. Please try again!"}})
	}

	c.Status(http.StatusCreated)
}
