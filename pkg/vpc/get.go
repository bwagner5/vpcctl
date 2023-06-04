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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/samber/lo"
)

func (v Client) getVPC(ctx context.Context, opts GetOptions) (*types.Vpc, error) {
	vpcOut, err := v.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{opts.Name},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(vpcOut.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC %s not found", opts.Name)
	}
	return &vpcOut.Vpcs[0], nil
}

func (v Client) getSubnets(ctx context.Context, vpcID string, _ GetOptions) ([]*types.Subnet, error) {
	subnetsOut, err := v.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return lo.Map(subnetsOut.Subnets, func(subnet types.Subnet, _ int) *types.Subnet { return &subnet }), nil
}

func (v Client) getRouteTables(ctx context.Context, vpcID string, _ GetOptions) ([]*types.RouteTable, error) {
	routeTablesOut, err := v.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return lo.Map(routeTablesOut.RouteTables, func(rt types.RouteTable, _ int) *types.RouteTable { return &rt }), nil
}

func (v Client) getIGW(ctx context.Context, vpcID string, _ GetOptions) (*types.InternetGateway, error) {
	igwOut, err := v.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(igwOut.InternetGateways) == 0 {
		return nil, nil
	}
	return &igwOut.InternetGateways[0], nil
}

func (v Client) getNATGW(ctx context.Context, vpcID string, _ GetOptions) (*types.NatGateway, error) {
	natGWOut, err := v.ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", CreatedByTagKey)),
				Values: []string{CreatedByTagValue},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(natGWOut.NatGateways) == 0 {
		return nil, nil
	}
	return &natGWOut.NatGateways[0], nil
}
