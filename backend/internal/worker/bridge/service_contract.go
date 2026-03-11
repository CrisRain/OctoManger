package bridge

type ServiceExecuteRequest struct {
	ScriptPath string `json:"script_path"`
	Input      Input  `json:"input"`
}

type ServiceExecuteResponse struct {
	Output *Output  `json:"output,omitempty"`
	Error  string   `json:"error,omitempty"`
	Logs   []string `json:"logs,omitempty"`
	Stdout string   `json:"stdout,omitempty"`
	Stderr string   `json:"stderr,omitempty"`
}
