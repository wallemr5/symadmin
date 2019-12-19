package controller

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/workload"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, workload.Add)
}
