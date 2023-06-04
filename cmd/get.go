/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/bwagner5/vpcctl/pkg/vpc"
)

const (
	OutputJSON   = "json"
	OutputEKSCTL = "eksctl"
)

type GetOptions struct {
	Name   string `yaml:"name"`
	Output string `yaml:"output"`
}

var (
	getOpts = GetOptions{}
	cmdGet  = &cobra.Command{
		Use:   "get [--name my-vpc]",
		Short: "Get a VPC",
		Long:  `Get a VPC with subresources like subnets, route-tables, etc.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			opts, err := ParseConfig(globalOpts, getOpts)
			if err != nil {
				fmt.Printf("Error parsing config file (%s): %s", globalOpts.ConfigFile, err)
			}
			if globalOpts.Verbose {
				fmt.Println(PrettyEncode(opts))
			}
			cfg, err := config.LoadDefaultConfig(cmd.Context())
			if err != nil {
				fmt.Printf("Error getting AWS config: %s", err)
				os.Exit(1)
			}

			vpcClient := vpc.New(cfg)
			vpcDetails, err := vpcClient.Get(cmd.Context(), vpc.GetOptions{Name: opts.Name})
			if err != nil {
				fmt.Println(PrettyEncode(vpcDetails))
				fmt.Println(err)
				os.Exit(2)
			}
			switch opts.Output {
			case OutputEKSCTL:
				fmt.Println(lo.Must(vpcDetails.OutputEKSCTL()))
			case OutputJSON:
				fmt.Println(PrettyEncode(vpcDetails))
			}
		},
	}
)

func init() {
	cmdGet.Flags().StringVarP(&getOpts.Name, "name", "n", "", "Name of the VPC")
	cmdGet.Flags().StringVarP(&getOpts.Output, "output", "o", OutputJSON, "Output format")
	rootCmd.AddCommand(cmdGet)
}
