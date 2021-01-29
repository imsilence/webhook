package requests

type Project struct {
	ID         int64  `json:"id"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Decription string `json:"description"`
	WebURL     string `json:"web_url"`
	GitSSHURL  string `json:"git_ssh_url"`
	GitHTTPURL string `json:"git_http_url"`
}
