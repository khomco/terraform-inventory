package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type meta struct {
	HostVars	map[string]map[string]string	`json:"hostvars"`
}

func gatherResources(states []*state) map[string]interface{} {
	metaData   := make(map[string]map[string]string, 0)
	groups    := make(map[string]interface{}, 0)

	for _, state := range states {
		for _, res := range state.resources() {
			for _, grp := range res.Groups() {

				_, ok := groups[grp]
				if !ok {
					groups[grp] = make(map[string][]string)
				}
				groups[grp].(map[string][]string)["hosts"] = append(groups[grp].(map[string][]string)["hosts"], res.Name())
			}
			metaData[res.Name()] = res.Attributes()
		}
		if len(state.outputs()) > 0 {
			groups["all"] = make(map[string]string, 0)
			for _, out := range state.outputs() {
				groups["all"].(map[string]string)[out.keyName] = out.value
			}
		}
	}
	groups["_meta"] = meta{HostVars: metaData}


	return groups
}

func cmdList(stdout io.Writer, stderr io.Writer, states []*state) int {
	return output(stdout, stderr, gatherResources(states))
}

func cmdHost(stdout io.Writer, stderr io.Writer, states []*state, hostname string) int {
	for _, state := range states {
		for _, res := range state.resources() {
			if hostname == res.Name() {
				return output(stdout, stderr, res.Attributes())
			}
		}
	}

	fmt.Fprintf(stderr, "No such host: %s\n", hostname)
	return 1
}

// output marshals an arbitrary JSON object and writes it to stdout, or writes
// an error to stderr, then returns the appropriate exit code.
func output(stdout io.Writer, stderr io.Writer, whatever interface{}) int {
	b, err := json.Marshal(whatever)
	if err != nil {
		fmt.Fprintf(stderr, "Error encoding JSON: %s\n", err)
		return 1
	}

	_, err = stdout.Write(b)
	if err != nil {
		fmt.Fprintf(stderr, "Error writing JSON: %s\n", err)
		return 1
	}

	return 0
}
