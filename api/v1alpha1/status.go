package v1alpha1

type CurrentStatus struct {
	State    string `json:"state,omitempty"`
	ErrorMsg string `json:"errorMsg"`
}

func NewCurrentStatus() *CurrentStatus {
	return &CurrentStatus{
		State:    "None",
		ErrorMsg: "",
	}
}

func (c *CurrentStatus) String() string {
	return c.State
}

func (c *CurrentStatus) SetCreated() {
	c.State = "Created"
}

func (c *CurrentStatus) SetRunning() {
	c.State = "Running"
}

func (c *CurrentStatus) SetError(err string) {
	c.State = "Error"
	c.ErrorMsg = err
}

func (c *CurrentStatus) SetTerminating() {
	c.State = "Terminating"
}

func (c *CurrentStatus) SetDeleted() {
	c.State = "Deleted"
}

func (c *CurrentStatus) SetNone() {
	c.State = "None"
}

func (c *CurrentStatus) IsNothing() bool {
	return c.State == ""
}
func (c *CurrentStatus) IsNone() bool {
	return c.State == "None"
}

func (c *CurrentStatus) IsCreated() bool {
	return c.State == "Created"
}

func (c *CurrentStatus) IsRunning() bool {
	return c.State == "Running"
}

func (c *CurrentStatus) IsError() bool {
	return c.State == "Error"
}

func (c *CurrentStatus) IsTerminating() bool {
	return c.State == "Terminating"
}

func (c *CurrentStatus) IsDeleted() bool {
	return c.State == "Deleted"
}

func (c *CurrentStatus) IsTerminated() bool {
	return c.State == "Terminating"
}
