package model

type RayJobResponse struct {
	Namespace string `json:"namespace"`
	JobID     string `json:"jobId"`
}

func NewRayJobResponse(namespace, jobID string) *RayJobResponse {
	return &RayJobResponse{
		Namespace: namespace,
		JobID:     jobID,
	}
}
