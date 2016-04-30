package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type meta struct {
	HostVars	map[string]map[string]string	`json:"hostvars"`
}

func gatherResources(states []*state) map[string]interface{} {
	metaData	:= make(map[string]map[string]string, 0)
	groups		:= make(map[string]interface{}, 0)

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

func cmdInventory(stdout io.Writer, stderr io.Writer, states []*state) int {
	groups := gatherResources(states)
	for group, res := range groups {
		// ignore _meta data
		if group == "_meta" {
			continue
		}
		_, err := io.WriteString(stdout, "["+group+"]\n")
		if err != nil {
			fmt.Fprintf(stderr, "Error writing Inventory: %s\n", err)
			return 1
		}

		groupvalue := reflect.ValueOf(res)
		for _, key := range groupvalue.MapKeys() {
			value := groupvalue.MapIndex(key)
			if value.Kind() == reflect.Slice {
				for _, ress := range res.(map[string][]string) {
					for _, host := range ress {
						_, err := io.WriteString(stdout, host+"\n")
						if err != nil {
							fmt.Fprintf(stderr, "Error writing Inventory: %s\n", err)
							return 1
						}
					}
				}

			} else if value.Kind() == reflect.String {
				_, err := io.WriteString(stdout, res.(map[string]string)[key.Interface().(string)]+"\n")
				if err != nil {
					fmt.Fprintf(stderr, "Error writing Inventory: %s\n", err)
					return 1
				}
			}
		}

		_, err = io.WriteString(stdout, "\n")
		if err != nil {
			fmt.Fprintf(stderr, "Error writing Inventory: %s\n", err)
			return 1
		}
	}

	return 0
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
