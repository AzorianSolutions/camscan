package workers

import (
	"as/camscan/internal/camscan/types"
	"context"
	"database/sql"
)

type JobID string
type JobType string
type JobMetadata map[string]interface{}

type ExecutionFn func(ctx context.Context, args interface{}, descriptor JobDescriptor) (interface{}, error)

type JobDescriptor struct {
	ID        JobID
	JType     JobType
	AppConfig types.AppConfig
	Metadata  map[string]interface{}
	Db        *sql.DB
}

type Result struct {
	Value      interface{}
	Err        error
	Descriptor JobDescriptor
}

type Job struct {
	Descriptor JobDescriptor
	ExecFn     ExecutionFn
	Args       interface{}
}

func (j Job) execute(ctx context.Context) Result {
	value, err := j.ExecFn(ctx, j.Args, j.Descriptor)
	if err != nil {
		return Result{
			Err:        err,
			Descriptor: j.Descriptor,
		}
	}

	return Result{
		Value:      value,
		Descriptor: j.Descriptor,
	}
}
