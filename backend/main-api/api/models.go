package api

type healthzOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}
