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

var (
	cmdList = &cobra.Command{
		Use:   "list ",
		Short: "List VPCs",
		Long:  `List VPCs created with vpcctl`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadDefaultConfig(cmd.Context())
			if err != nil {
				fmt.Printf("Error getting AWS config: %s", err)
				os.Exit(1)
			}

			vpcClient := vpc.New(cfg)
			vpcs, err := vpcClient.List(cmd.Context())
			if err != nil {
				fmt.Println(err)
				os.Exit(2)
			}
			for _, v := range vpcs {
				fmt.Println(v)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(cmdList)
}
