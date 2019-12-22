# check_eve_ng
[![Go Report Card](https://goreportcard.com/badge/github.com/inexio/check_eve_ng)](https://goreportcard.com/report/github.com/inexio/check_eve_ng)
[![GitHub license](https://img.shields.io/badge/license-BSD-blue.svg)](https://github.com/inexio/check_eve_ng/blob/master/LICENSE)

## Description
monitoring check plugin for the [EVE-NG](https://www.eve-ng.net/) [API](https://www.eve-ng.net/index.php/documentation/howtos/how-to-eve-ng-api/) (written in golang). The plugin complies with the [Monitoring Plugins Development Guidelines](https://www.monitoring-plugins.org/doc/guidelines.html) and should therefore be compatible with [nagios](https://www.nagios.org/), [icinga2](https://icinga.com/), [zabbix](https://www.zabbix.com/), [checkmk](https://checkmk.com/), etc.

## Example / Usage

	Usage:
	  check_eve_ng [OPTIONS]

	Application Options:
	      --hostname=                    Hostname
	      --username=                    Username
	      --password=                    Password
	      --lab=                         Lab that will be included in monitoring
	      --all-nodes-up                 Check if all nodes in the given labs are up
	      --performance-data-json-label  Output performance data label in json format
	      --exclude-node=                Exclude a node by its uuid
	      --labs-exist                   Check if all given labs exist (only checks for implicit named labs in the input parameters)
	      --lab-performance-data         Print performance data for all included labs
	      --force-http                   Force http instead of https

	Help Options:
	  -h, --help                         Show this help message
	
	
	
	Examples:
	Check if a lab exists:
	./check_eve_ng --hostname $hostname --username $username --password $password --lab myLab --labs-exist 
	CRITICAL: lab myLab does not exist! | 'vpcs'=0;;;; 'iol'=0;;;; 'dynamips'=0;;;; 'qemu'=4;;;; 'docker'=0;;;;

	Check if all nodes are up on all labs:
	./check_eve_ng --hostname $hostname --username $username --password $password --lab all --all-hosts-up
	OK: checked!


## Installation

To install, use `go get` and `go build`:

    go get github.com/inexio/check_eve_ng
    go build github.com/inexio/check_eve_ng

## Staying up to date

To update to the latest version, use `go get -u github.com/inexio/check_eve_ng`.
