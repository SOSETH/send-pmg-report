/*
 * Copyright (C) 2018  Maximilian Falkenstein <mfalkenstein@sos.ethz.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */
package main

import (
	"flag"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type CFG struct {
	RedirectedDomains []struct {
		Domain string `yaml:"domain"`
		Destination string `yaml:"destination"`
	} `yaml:"redirectedDomains"`
	RedirectedTargets []struct {
		Target string `yaml:"target"`
		Destination string `yaml:"destination"`
	} `yaml:"redirectedTargets"`
}

func pmgQmFields(c rune) bool {
	return c == ' '
}

func main() {
	fileName := flag.String("from-file", "", "Read pmgqm output from file. Useful for debugging.")
	sudo := flag.Bool("sudo", true, "Use sudo to execute pmgqm")
	timespan := flag.String("timespan", "week", "Which timespan to analyze (see pmgqm manpage for details")
	cfglocation := flag.String("config", "config.yaml", "Path to configuration file")
	send := flag.Bool("exec", false, "Whether to actually send reports")
	flag.Parse()

	cfg := CFG{}

	if contents, err := ioutil.ReadFile(*cfglocation); err == nil {
		err = yaml.UnmarshalStrict(contents, &cfg)
		if err != nil {
			fmt.Println("ERROR: Couldn't parse the configuration file!", err)
			os.Exit(-1)
		}
	} else {
		fmt.Println("ERROR: Couldn't read the configuration file!", err)
		os.Exit(-1)
	}

	// No set for you
	blacklistedUser := make(map[string]string)
	blacklistedDomain := make(map[string]string)
	for _, user := range cfg.RedirectedTargets {
		blacklistedUser[user.Target] = user.Destination
	}
	for _, domain := range cfg.RedirectedDomains {
		blacklistedDomain[domain.Domain] = domain.Destination
	}

	var output []string
	if *fileName != "" {
		if f, err := os.Open(*fileName); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				output = append(output, scanner.Text())
			}
		} else {
			fmt.Println("ERROR: Couldn't read the specified file!", err)
			os.Exit(-1)
		}
	} else {
		// Run pmgqm directly
		var pmgqmCMD *exec.Cmd
		if *sudo {
			pmgqmCMD = exec.Command("sudo", "pmgqm", "status", "-timespan", *timespan)
		} else {
			pmgqmCMD = exec.Command("pmgqm", "status", "-timespan", *timespan)
		}
		if pmgqmOut, err := pmgqmCMD.Output(); err != nil {
			fmt.Println("ERROR: Couldn't execute pmgqm!", err)
			os.Exit(-1)
		} else {
			output = strings.Split(string(pmgqmOut[:]), "\n")
		}
	}

	for _, line := range output {
		fields := strings.FieldsFunc(line, pmgQmFields)
		if len(fields) == 3 && strings.Contains(fields[2], "@") {
			mailParts := strings.Split(fields[2], "@")
			args := []string{
				"pmgqm",
				"send",
				"-receiver",
				fields[2],
				"-timespan",
				*timespan,
			}
			if target, err := blacklistedUser[fields[2]]; err {
				fmt.Printf("Redirect line %q, user is blacklisted\n", fields)
				args = append(args, "-redirect")
				args = append(args, target)
			} else if target, err := blacklistedDomain[mailParts[1]]; err {
				fmt.Printf("Redirect line %q, domain is blacklisted\n", fields)
				args = append(args, "-redirect")
				args = append(args, target)
			}
			fmt.Printf("%q \n", args)
			if *send {
				fmt.Printf("Sending report for %q \n", fields[2])
				var cmd *exec.Cmd
				if *sudo {
					cmd = exec.Command("sudo", args...)
				} else {
					cmd = exec.Command("pmgqm", args[1:]...)
				}
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			}
		}
	}
}
