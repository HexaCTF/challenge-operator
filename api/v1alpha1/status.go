package v1alpha1

type CurrentStatus struct {
	State    string `json:"state,omitempty"`
	ErrorMsg string `json:"errorMsg"`
}

func (c *CurrentStatus) String() string {
	return c.State
}
