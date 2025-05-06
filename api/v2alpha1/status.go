package v2alpha1

type CurrentStatus struct {
	Status string `json:"status,omitempty"`
}

func NewCurrentStatus() *CurrentStatus {
	return &CurrentStatus{
		Status: "None",
	}
}

func (c *CurrentStatus) Pending() {
	c.Status = "Pending"
}

func (c *CurrentStatus) Running() {
	c.Status = "Running"
}

func (c *CurrentStatus) Terminating() {
	c.Status = "Terminating"
}

func (c *CurrentStatus) Deleted() {
	c.Status = "Deleted"
}
