package main

import (
	"check_eve_ng/evengapi"
	"fmt"
	"github.com/inexio/go-monitoringplugin"
	"github.com/jessevdk/go-flags"
	"github.com/oleiade/reflections"
	"github.com/pkg/errors"
	"os"
	"regexp"
	"strings"
)

func main() {
	opts, err := parseArgs()
	if err != nil {
		os.Exit(3) //parseArgs() prints errors to stdout
	}
	err = validateArgs(opts)
	if err != nil {
		fmt.Println("Invalid arguments: " + err.Error())
		os.Exit(3)
	}

	response := monitoringplugin.NewResponse("checked")
	defer response.OutputAndExit()

	if opts.PerformanceDataJSONLabel {
		response.SetPerformanceDataJsonLabel(true)
	}

	eveNgAPI, err := evengapi.NewEveNgApi(opts.Hostname, opts.Username, opts.Password)
	if err != nil {
		response.UpdateStatus(monitoringplugin.UNKNOWN, "error during creating new eve ng api: "+err.Error())
		return
	}
	if opts.ForceHTTP {
		err = eveNgAPI.ForceHttp(true)
		if err != nil {
			response.UpdateStatus(monitoringplugin.UNKNOWN, "eve ng api might not have been created properly: "+err.Error())
			return
		}
	}
	err = eveNgAPI.Login()
	if err != nil {
		response.UpdateStatus(monitoringplugin.UNKNOWN, "error during Login: "+err.Error())
		return
	}
	defer func() {
		err = eveNgAPI.Logout()
		if err != nil {
			response.UpdateStatus(monitoringplugin.UNKNOWN, "error during logout: "+err.Error())
		}
	}()

	//System Status
	statusMetrics := []string{"iol", "dynamips", "qemu", "docker", "vpcs"}
	systemStatus, err := eveNgAPI.GetSystemStatus()
	if err != nil {
		response.UpdateStatus(monitoringplugin.UNKNOWN, "error during Get System Status: "+err.Error())
	}

	for _, metric := range statusMetrics {
		valueInterface, err := reflections.GetField(systemStatus, strings.Title(metric))
		if err != nil {
			response.UpdateStatus(monitoringplugin.UNKNOWN, "error while getting field from system status struct: "+err.Error())
			continue
		}
		value, ok := valueInterface.(*float64)
		if !ok {
			response.UpdateStatus(monitoringplugin.UNKNOWN, "error while converting interface to *int")
			continue
		}
		err = response.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint(metric, *value, ""))
		if err != nil {
			response.UpdateStatus(monitoringplugin.UNKNOWN, "error while adding new performance data point: "+err.Error())
		}
	}

	labs, err := getLabs(&opts.Labs, eveNgAPI)
	if err != nil {
		response.UpdateStatus(monitoringplugin.UNKNOWN, "error during getLabs: "+err.Error())
		return
	}

	//fmt.Println(labs)

	for _, lab := range labs {
		result, err :=
			eveNgAPI.GetAllNodesForLab(lab)
		if err != nil {
			match, errRegex := regexp.MatchString(`Lab does not exist`, err.Error())
			if errRegex != nil {
				response.UpdateStatus(monitoringplugin.UNKNOWN, "regex error: "+errRegex.Error())
				continue
			}
			if match && opts.LabsExist {
				response.UpdateStatus(monitoringplugin.CRITICAL, "lab "+lab+" does not exist!")
				continue
			}
			response.UpdateStatus(monitoringplugin.UNKNOWN, "error during request for all nodes for lab "+lab+": "+err.Error())
		} else {
			labHostsUp := 0
			labHostsDown := 0
			for _, nodeData := range result {
				if nodeData.Status == 0 {
					labHostsDown++
					if opts.AllNodesUp && !inArray(nodeData.Uuid, opts.ExcludeNode) {
						response.UpdateStatus(monitoringplugin.CRITICAL, "node "+nodeData.Name+" ("+nodeData.Image+") in lab "+lab+" is down! (uuid: "+nodeData.Uuid+")")
					}
				} else {
					labHostsUp++
				}
			}

			if opts.LabPerformanceData {
				err := response.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("nodes_up", float64(labHostsUp), "").SetLabelTag(lab))
				if err != nil {
					response.UpdateStatus(monitoringplugin.UNKNOWN, "error during add performance data point: "+err.Error())
				}
				err = response.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("nodes_down", float64(labHostsDown), "").SetLabelTag(lab))
				if err != nil {
					response.UpdateStatus(monitoringplugin.UNKNOWN, "error during add performance data point: "+err.Error())
				}
			}
		}
	}
	return
}

type cliOpts struct {
	Hostname                 string   `long:"hostname" description:"Hostname" required:"true"`
	Username                 string   `long:"username" description:"Username" required:"true"`
	Password                 string   `long:"password" description:"Password" required:"true"`
	Labs                     []string `long:"lab" description:"Lab that will be included in monitoring" required:"false"`
	AllNodesUp               bool     `long:"all-nodes-up" description:"Check if all nodes in the given labs are up" required:"false"`
	PerformanceDataJSONLabel bool     `long:"performance-data-json-label" description:"Output performance data label in json format" required:"false"`
	ExcludeNode              []string `long:"exclude-node" description:"Exclude a node by its uuid" required:"false"`
	LabsExist                bool     `long:"labs-exist" description:"Check if all given labs exist (only checks for implicit named labs in the input parameters)" required:"false"`
	LabPerformanceData       bool     `long:"lab-performance-data" description:"Print performance data for all included labs" required:"false"`
	ForceHTTP                bool     `long:"force-http" description:"Force http instead of https" required:"false"`
}

func parseArgs() (*cliOpts, error) {
	opts := &cliOpts{}
	_, err := flags.Parse(opts)
	return opts, err
}

func validateArgs(opts *cliOpts) error {
	if (len(opts.Labs) == 0) && (opts.LabsExist || opts.LabPerformanceData) {
		return errors.New("the options --labs-exist and --lab-performance-data cannot be used when there are no given labs")
	}
	if (len(opts.Labs) == 1) && opts.Labs[0] == "all" && (opts.LabsExist) {
		return errors.New("the options --labs-exist cannot be used when there is no specific given lab to check for existence")
	}
	if len(opts.Labs) > 0 && !(opts.LabsExist || opts.LabPerformanceData || opts.AllNodesUp) {
		return errors.New("there are labs defined but no monitoring modes for them (like --all-nodes-up, --labs-exist etc.)")
	}
	return nil
}

func getLabs(optLabs *[]string, eveNgAPI *evengapi.EveNgApi) ([]string, error) {
	useAllLabs := false
	for i, lab := range *optLabs {
		if lab == "all" {
			useAllLabs = true
			(*optLabs)[i] = (*optLabs)[len(*optLabs)-1]
			*optLabs = (*optLabs)[:len(*optLabs)-1]
			break
		}
	}

	if useAllLabs {
		labs, err := eveNgAPI.GetAllLabs()
		if err != nil {
			return nil, errors.Wrap(err, "error during GetAllLabs")
		}
		return evengapi.SliceMerge(labs, *optLabs), nil
	}
	return *optLabs, nil
}

func inArray(search string, arr []string) bool {
	for _, i := range arr {
		if i == search {
			return true
		}
	}
	return false
}
