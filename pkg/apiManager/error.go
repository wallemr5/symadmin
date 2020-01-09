package apiManager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
)

// Defined error code
const (
	// BaseError
	ParamInvalidError   = 1001
	RecordNotExistError = 1002

	// PodError
	GetPodError      = 2001
	GetPodEventError = 2002

	// DeploymentError
	GetDeploymentError = 3001

	// ServiceError
	GetServiceError = 4001

	// TerminalError
	GetTerminalError    = 5001
	WebsocketError      = 5002
	RequestK8sExecError = 5003

	// OtherError
	GetClusterError  = 9001
	GetEndpointError = 9002
	GetNodeError     = 9003
)

// AbortHTTPError ...
func AbortHTTPError(c *gin.Context, code int, msg string, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest,
		model.ErrorResponse{
			Code:    code,
			Message: msg,
			Error:   err.Error(),
		})
}
