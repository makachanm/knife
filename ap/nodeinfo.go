package ap

// NodeInfo represents the root NodeInfo object
type NodeInfo struct {
	Version          string            `json:"version"`
	Software         NodeInfoSoftware  `json:"software"`
	Protocols        []string          `json:"protocols"`
	Services         NodeInfoServices  `json:"services"`
	OpenRegistrations bool              `json:"openRegistrations"`
	Usage            NodeInfoUsage     `json:"usage"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// NodeInfoSoftware represents the software information
type NodeInfoSoftware struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Additional fields like `repository` and `homepage` can be added if needed.
	Repository string `json:"repository,omitempty"`
	Homepage   string `json:"homepage,omitempty"`
}

// NodeInfoServices represents the services information
type NodeInfoServices struct {
	Outbound []string `json:"outbound"`
	Inbound  []string `json:"inbound"`
}

// NodeInfoUsage represents the usage statistics
type NodeInfoUsage struct {
	Users NodeInfoUsageUsers `json:"users"`
	// Additional fields like `localPosts` and `localComments` can be added if available.
}

// NodeInfoUsageUsers represents user statistics
type NodeInfoUsageUsers struct {
	Total          int `json:"total"`
	ActiveHalfyear int `json:"activeHalfyear"`
	ActiveMonth    int `json:"activeMonth"`
}
