package evengapi

type systemStatusResponse struct {
	Data SystemStatus
}

//SystemStatus contains the status information returned by the system status api call
type SystemStatus struct {
	Dynamips *float64
	Vpcs     *float64
	Docker   *float64
	Qemu     *float64
	Iol      *float64
}

type folderResponse struct {
	Code    int
	Data    folderData
	Message string
	Status  string
}

type folderData struct {
	Folders []folder
	Labs    []lab
}
type folder struct {
	Name string
	Path string
}

type lab struct {
	File string
	Path string
}

type nodesResponse struct {
	Data map[string]Nodes
}

//Nodes contains necessary information on a node
type Nodes struct {
	Name   string
	Status int
	UUID   string
	Image  string
}
