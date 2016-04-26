package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// keyNames contains the names of the keys to check for in each resource in the
// state file. This allows us to support multiple types of resource without too
// much fuss.
var keyNames []string
var nameParser *regexp.Regexp

func init() {
	keyNames = []string{
		"ipv4_address",                                        // DO
		"public_ip",                                           // AWS
		"private_ip",                                          // AWS
		"ipaddress",                                           // CS
		"ip_address",                                          // VMware
		"access_ip_v4",                                        // OpenStack
		"floating_ip",                                         // OpenStack
		"network_interface.0.access_config.0.nat_ip",          // GCE
		"network_interface.0.access_config.0.assigned_nat_ip", // GCE
		"network_interface.0.address",                         // GCE
		"private_ip_address",																	 // CLC
	}

	// type.name.0
	nameParser = regexp.MustCompile(`^(\w+)\.([\w\-]+)(?:\.(\d+))?$`)
}

type Resource struct {

	// The state (as unmarshalled from the statefile) which this resource wraps.
	// Everything which Terraform knows about the resource can be found in here.
	State resourceState

	// The key name of the resource, provided to the constructor. Unfortunately,
	// it seems like the counter index can only be found here.
	keyName string

	// Extracted from keyName
	resourceType string
	baseName     string
	counter      int
}

func NewResource(keyName string, state resourceState) (*Resource, error) {
	m := nameParser.FindStringSubmatch(keyName)

	// This should not happen unless our regex changes.
	// TODO: Warn instead of silently ignore error?
	if len(m) != 4 {
		return nil, fmt.Errorf("couldn't parse keyName: %s", keyName)
	}

	var c int
	var err error
	if m[3] != "" {

		// The third section should be the index, if it's present. Not sure what
		// else we can do other than panic (which seems highly undesirable) if that
		// isn't the case.
		c, err = strconv.Atoi(m[3])
		if err != nil {
			return nil, err
		}
	}

	return &Resource{
		State:        state,
		keyName:      keyName,
		resourceType: m[1],
		baseName:     m[2],
		counter:      c,
	}, nil
}

func (r Resource) Name() string {
	return r.Attributes()["id"]
}

func (r Resource) IsSupported() bool {
	return r.Address() != ""
}

// Groups returns the list of Ansible groups which this resource should be
// included in.
func (r Resource) Groups() []string {
	groups := []string{
		r.baseName,
		fmt.Sprintf("type_%s", r.resourceType),
	}

	switch r.resourceType {
	case "clc_server":
		groups = append(groups, "datacenter_" + strings.ToLower(r.Name()[:3]))
	}

	for k, v := range r.Tags() {
		g := fmt.Sprintf("%s_%s", k, v)
		groups = append(groups, g)
	}

	return groups
}

// Tags returns a map of arbitrary key/value pairs explicitly associated with
// the resource. Different providers have different mechanisms for attaching
// these.
func (r Resource) Tags() map[string]string {
	t := map[string]string{}

	switch r.resourceType {
	case "aws_instance":
		for k, v := range r.Attributes() {
			parts := strings.SplitN(k, ".", 2)
			if len(parts) == 2 && parts[0] == "tags" && parts[1] != "#" {
				kk := strings.ToLower(parts[1])
				vv := strings.ToLower(v)
				t[kk] = vv
			}
		}
	}

	return t
}

// Attributes returns a map containing everything we know about this resource.
func (r Resource) Attributes() map[string]string {
	raw := r.State.Primary.Attributes

	switch r.resourceType {
	case "clc_server":
		clcServer := make(map[string]string)
		clcServer["id"] = raw["id"]
		clcServer["cpu"] = raw["cpu"]
		clcServer["memory_mb"] = raw["memory_mb"]
		clcServer["name"] = raw["name"]
		clcServer["type"] = raw["type"]
		clcServer["storage_type"] = raw["storage_type"]
		clcServer["ansible_ssh_host"] = raw["private_ip_address"]
		clcServer["private_ip_address"] = raw["private_ip_address"]
		clcServer["ansible_ssh_user"] = "root"
		clcServer["ansible_ssh_pass"] = raw["password"]
		clcServer["datacenter"] = strings.ToUpper(raw["id"][:3])
		return clcServer
	default:
		return raw
	}
}

// NameWithCounter returns the resource name with its counter. For resources
// created without a 'count=' attribute, this will always be zero.
func (r Resource) NameWithCounter() string {
	return fmt.Sprintf("%s.%d", r.baseName, r.counter)
}

// Address returns the IP address of this resource.
func (r Resource) Address() string {
	if keyName := os.Getenv("TF_KEY_NAME"); keyName != "" {
		if ip := r.State.Primary.Attributes[keyName]; ip != "" {
			return ip
		}
	} else {
		for _, key := range keyNames {
			if ip := r.State.Primary.Attributes[key]; ip != "" {
				return ip
			}
		}
	}

	return ""
}
