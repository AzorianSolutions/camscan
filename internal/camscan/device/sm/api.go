package sm

import (
	"as/camscan/internal/camscan/workers"
	"context"
)

func ScanDevice(ctx context.Context, args interface{}, descriptor workers.JobDescriptor) (interface{}, error) {
	result := make(map[string]interface{})
	return result, nil
}
