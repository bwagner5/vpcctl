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
	"math/rand"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/bwagner5/vpcctl/pkg/vpc"
)

type CreateOptions struct {
	Name    string            `yaml:"name"`
	CIDR    string            `yaml:"cidr"`
	Subnets []SubnetOptions   `yaml:"subnets"`
	Tags    map[string]string `yaml:"tags"`
}

type SubnetOptions struct {
	AZ     string `yaml:"az"`
	CIDR   string `yaml:"cidr"`
	Public bool   `yaml:"public"`
}

var (
	createOpts = CreateOptions{}
	cmdCreate  = &cobra.Command{
		Use:   "create [--name my-vpc]",
		Short: "Create a VPC",
		Long:  `Create a VPC with subresources like subnets, route-tables, etc. to get going quickly`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, _ []string) {
			opts, err := ParseConfig(globalOpts, createOpts)
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
			vpcDetails, err := vpcClient.Create(cmd.Context(), CreateCLIOptsToVPCOpts(opts))
			if err != nil {
				fmt.Println(PrettyEncode(vpcDetails))
				fmt.Println(err)
				os.Exit(2)
			}
			fmt.Println(PrettyEncode(vpcDetails))
		},
	}
)

func init() {
	//nolint:gosec // we don't need to use crypto/rand here for a default name
	cmdCreate.Flags().StringVarP(&createOpts.Name, "name", "n", fmt.Sprintf("vpcctl-generated-%d", rand.Int()), "Name of the VPC")
	cmdCreate.Flags().StringVarP(&createOpts.CIDR, "cidr", "c", "10.0.0.0/16", "CIDR of the VPC")
	cmdCreate.Flags().StringToStringVarP(&createOpts.Tags, "tags", "t", nil, "Additional tags to add to VPC resources")
	rootCmd.AddCommand(cmdCreate)
}

func CreateCLIOptsToVPCOpts(opts CreateOptions) vpc.CreateOptions {
	return vpc.CreateOptions{
		Name: opts.Name,
		CIDR: opts.CIDR,
		Tags: opts.Tags,
		Subnets: lo.Map(opts.Subnets, func(snOpts SubnetOptions, _ int) vpc.CreateSubnetOptions {
			return vpc.CreateSubnetOptions{
				AZ:     snOpts.AZ,
				CIDR:   snOpts.CIDR,
				Public: snOpts.Public,
			}
		}),
	}
}
