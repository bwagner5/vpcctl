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

package vpc

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/samber/lo"
)

const (
	SubnetTypePublic  = "PUBLIC"
	SubnetTypePrivate = "PRIVATE"
	CreatedByTagKey   = "CreatedBy"
	CreatedByTagValue = "vpcctl"
)

var (
	defaultTags = []types.Tag{
		{Key: aws.String(CreatedByTagKey), Value: aws.String(CreatedByTagValue)},
	}
)

type Client struct {
	cfg       aws.Config
	ec2Client *ec2.Client
}

type CreateOptions struct {
	Name    string
	CIDR    string
	Subnets []CreateSubnetOptions
}

type DeleteOptions struct {
	Name                   string
	DeleteUnownedResources bool
}

type GetOptions struct {
	Name string
}

type CreateSubnetOptions struct {
	AZ     string
	CIDR   string
	Public bool
}

type Details struct {
	VPC             *types.Vpc
	Subnets         []*types.Subnet
	RouteTables     []*types.RouteTable
	InternetGateway *types.InternetGateway
	NATGateway      *types.NatGateway
}

func New(cfg aws.Config) *Client {
	return &Client{
		cfg:       cfg,
		ec2Client: ec2.NewFromConfig(cfg),
	}
}

// DefaultSubnets uses 3 subnets in the region with
// Private /18 CIDRs (16,382 IPs)
// Public /20 CIDRs (4,094 IPs)
func DefaultSubnets(region string) []CreateSubnetOptions {
	return []CreateSubnetOptions{
		{
			AZ:     fmt.Sprintf("%sa", region),
			CIDR:   "10.0.0.0/18",
			Public: false,
		},
		{
			AZ:     fmt.Sprintf("%sb", region),
			CIDR:   "10.0.64.0/18",
			Public: false,
		},
		{
			AZ:     fmt.Sprintf("%sc", region),
			CIDR:   "10.0.128.0/18",
			Public: false,
		},
		{
			AZ:     fmt.Sprintf("%sa", region),
			CIDR:   "10.0.192.0/20",
			Public: true,
		},
		{
			AZ:     fmt.Sprintf("%sb", region),
			CIDR:   "10.0.208.0/20",
			Public: true,
		},
		{
			AZ:     fmt.Sprintf("%sc", region),
			CIDR:   "10.0.224.0/20",
			Public: true,
		},
	}
}

func (v Client) List(ctx context.Context) ([]string, error) {
	vpcOut, err := v.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{"Name"},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return lo.Compact(lo.Map(vpcOut.Vpcs, func(vpc types.Vpc, _ int) string {
		if tag, ok := lo.Find(vpc.Tags, func(tag types.Tag) bool { return *tag.Key == "Name" }); ok {
			return *tag.Value
		}
		return ""
	})), nil
}

func (v Client) Create(ctx context.Context, opts CreateOptions) (*Details, error) {
	vpcDetails := &Details{}
	if len(opts.Subnets) == 0 {
		opts.Subnets = DefaultSubnets(v.cfg.Region)
	}
	log.Printf("Creating VPC %s", opts.Name)
	vpc, err := v.createVPC(ctx, opts)
	vpcDetails.VPC = vpc
	if err != nil {
		return vpcDetails, err
	}
	log.Printf("Created VPC %s", *vpc.VpcId)

	log.Println("Creating Subnets")
	subnets, err := v.createSubnets(ctx, *vpc.VpcId, opts)
	vpcDetails.Subnets = subnets
	if err != nil {
		return vpcDetails, err
	}
	privateSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return !*subnet.MapPublicIpOnLaunch })
	publicSubnets := lo.Filter(subnets, func(subnet *types.Subnet, _ int) bool { return *subnet.MapPublicIpOnLaunch })
	log.Printf("Created Public Subnets %s", lo.Map(publicSubnets, func(subnet *types.Subnet, _ int) string { return *subnet.SubnetId }))
	log.Printf("Created Private Subnets %s", lo.Map(privateSubnets, func(subnet *types.Subnet, _ int) string { return *subnet.SubnetId }))

	log.Println("Creating Route Tables")
	routeTables, err := v.createRouteTables(ctx, subnets, opts)
	vpcDetails.RouteTables = lo.Values(routeTables)
	if err != nil {
		return vpcDetails, err
	}
	log.Printf("Created Route Tables: %s", lo.Map(vpcDetails.RouteTables, func(rt *types.RouteTable, _ int) string { return *rt.RouteTableId }))

	log.Println("Creating Internet Gateway")
	igw, err := v.createIGW(ctx, *vpc.VpcId, routeTables[SubnetTypePublic], opts)
	vpcDetails.InternetGateway = igw
	if err != nil {
		return vpcDetails, err
	}
	log.Printf("Created Internet Gateway: %s", *vpcDetails.InternetGateway.InternetGatewayId)

	log.Println("Creating NAT Gateway")
	natGW, err := v.createNATGW(ctx, subnets, routeTables[SubnetTypePrivate], opts)
	vpcDetails.NATGateway = natGW
	if err != nil {
		return vpcDetails, err
	}
	log.Printf("Created NAT Gateway: %s", *vpcDetails.NATGateway.NatGatewayId)
	return vpcDetails, nil
}

func (v Client) Delete(ctx context.Context, opts DeleteOptions) (*Details, error) {
	log.Printf("Fetching VPC details for %s", opts.Name)
	vpcDetails, err := v.Get(ctx, GetOptions{Name: opts.Name})
	if err != nil {
		return vpcDetails, err
	}
	if vpcDetails.NATGateway != nil {
		log.Printf("Deleting NAT Gateway %s", *vpcDetails.NATGateway.NatGatewayId)
		if err := v.deleteNATGW(ctx, vpcDetails, opts); err != nil {
			return vpcDetails, err
		}
		log.Printf("Deleted NAT Gateway %s", *vpcDetails.NATGateway.NatGatewayId)
	}
	if vpcDetails.InternetGateway != nil {
		log.Printf("Deleting Internet Gateway %s", *vpcDetails.InternetGateway.InternetGatewayId)
		if err := v.deleteIGW(ctx, vpcDetails, opts); err != nil {
			return vpcDetails, err
		}
		log.Printf("Deleted Internet Gateway %s", *vpcDetails.InternetGateway.InternetGatewayId)
	}
	if len(vpcDetails.RouteTables) != 0 {
		log.Printf("Deleting Route Tables %v", lo.Map(vpcDetails.RouteTables, func(rt *types.RouteTable, _ int) string { return *rt.RouteTableId }))
		if err := v.deleteRouteTables(ctx, vpcDetails, opts); err != nil {
			return vpcDetails, err
		}
		log.Printf("Deleted Route Tables %v", lo.Map(vpcDetails.RouteTables, func(rt *types.RouteTable, _ int) string { return *rt.RouteTableId }))
	}
	if len(vpcDetails.Subnets) != 0 {
		log.Printf("Deleting Subnets %v", lo.Map(vpcDetails.Subnets, func(subnet *types.Subnet, _ int) string { return *subnet.SubnetId }))
		if err := v.deleteSubnets(ctx, vpcDetails, opts); err != nil {
			return vpcDetails, err
		}
		log.Printf("Deleted Subnets %v", lo.Map(vpcDetails.Subnets, func(subnet *types.Subnet, _ int) string { return *subnet.SubnetId }))
	}
	if vpcDetails.VPC != nil {
		log.Printf("Deleting VPC %s", *vpcDetails.VPC.VpcId)
		if err := v.deleteVPC(ctx, vpcDetails, opts); err != nil {
			return vpcDetails, err
		}
	}
	return vpcDetails, nil
}

func (v Client) Get(ctx context.Context, opts GetOptions) (*Details, error) {
	vpcDetails := &Details{}
	vpc, err := v.getVPC(ctx, opts)
	vpcDetails.VPC = vpc
	if err != nil {
		return vpcDetails, err
	}

	subnets, err := v.getSubnets(ctx, *vpc.VpcId, opts)
	vpcDetails.Subnets = subnets
	if err != nil {
		return vpcDetails, err
	}

	routeTables, err := v.getRouteTables(ctx, *vpc.VpcId, opts)
	vpcDetails.RouteTables = routeTables
	if err != nil {
		return vpcDetails, err
	}

	igw, err := v.getIGW(ctx, *vpc.VpcId, opts)
	vpcDetails.InternetGateway = igw
	if err != nil {
		return vpcDetails, err
	}

	natGW, err := v.getNATGW(ctx, *vpc.VpcId, opts)
	vpcDetails.NATGateway = natGW
	if err != nil {
		return vpcDetails, err
	}
	return vpcDetails, nil
}
