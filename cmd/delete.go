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
	"github.com/spf13/cobra"

	"github.com/bwagner5/vpcctl/pkg/vpc"
)

type DeleteOptions struct {
	Name string `yaml:"name"`
}

var (
	deleteOpts = DeleteOptions{}
	cmdDelete  = &cobra.Command{
		Use:   "delete [--name my-vpc]",
		Short: "Delete a VPC",
		Long:  `Delete a VPC with subresources like subnets, route-tables, etc.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, _ []string) {
			opts, err := ParseConfig(globalOpts, deleteOpts)
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
			vpcDetails, err := vpcClient.Delete(cmd.Context(), vpc.DeleteOptions{Name: opts.Name})
			if err != nil {
				fmt.Println(PrettyEncode(vpcDetails))
				fmt.Println(err)
				os.Exit(2)
			}
			fmt.Printf("Deleted VPC %s", opts.Name)
		},
	}
)

func init() {
	cmdDelete.Flags().StringVarP(&deleteOpts.Name, "name", "n", "", "Name of the VPC")
	rootCmd.AddCommand(cmdDelete)
}
