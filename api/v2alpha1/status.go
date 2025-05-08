package v2alpha1

type CurrentStatus struct {
	Status string `json:"status,omitempty"`
}

func NewCurrentStatus() *CurrentStatus {
	return &CurrentStatus{
		Status: "Pending",
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

func (c *CurrentStatus) Error(err error) {
	c.Status = "Error"
}

func (c *CurrentStatus) IsPending() bool {
	return c.Status == "Pending"
}

func (c *CurrentStatus) IsRunning() bool {
	return c.Status == "Running"
}

func (c *CurrentStatus) IsTerminating() bool {
	return c.Status == "Terminating"
}

func (c *CurrentStatus) IsDeleted() bool {
	return c.Status == "Deleted"
}
